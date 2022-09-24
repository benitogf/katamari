package stream

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"

	"github.com/benitogf/jsonpatch"

	"github.com/benitogf/coat"
	"github.com/gorilla/websocket"
)

const timeout = 15 * time.Second

// Subscribe : monitoring or filtering of subscriptions
type Subscribe func(key string) error

// Unsubscribe : function callback on subscription closing
type Unsubscribe func(key string)

type GetFn func(key string) ([]byte, error)

type EncodeFn func(data []byte) string

// Conn extends the websocket connection with a mutex
// https://godoc.org/github.com/gorilla/websocket#hdr-Concurrency
type Conn struct {
	mutex sync.Mutex
	conn  *websocket.Conn
}

// Pool of key filtered connections
type Pool struct {
	mutex       sync.RWMutex
	Key         string
	cache       Cache
	connections []*Conn
}

// Stream a group of pools
type Stream struct {
	mutex         sync.RWMutex
	OnSubscribe   Subscribe
	OnUnsubscribe Unsubscribe
	ForcePatch    bool
	pools         []*Pool
	Console       *coat.Console
}

type BroadcastOpt struct {
	Get      GetFn
	Encode   EncodeFn
	Callback func()
}

// Cache holds version and data
type Cache struct {
	Version int64
	Data    []byte
}

var StreamUpgrader = websocket.Upgrader{
	// define the upgrade success
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Upgrade") == "websocket"
	},
	Subprotocols: []string{"bearer"},
}

func (sm *Stream) findPool(key string) int {
	poolIndex := -1
	for i := range sm.pools {
		if sm.pools[i].Key == key {
			poolIndex = i
			break
		}
	}

	return poolIndex
}

func (sm *Stream) InitClock() {
	if len(sm.pools) == 0 {
		sm.pools = append(
			sm.pools,
			&Pool{Key: ""})
	}
}

// New stream on a key
func (sm *Stream) New(key string, w http.ResponseWriter, r *http.Request) (*Conn, error) {
	wsClient, err := StreamUpgrader.Upgrade(w, r, nil)

	if err != nil {
		sm.Console.Err("socketUpgradeError["+key+"]", err)
		return nil, err
	}

	err = sm.OnSubscribe(key)
	if err != nil {
		return nil, err
	}

	return sm.new(key, wsClient), nil
}

// Open a connection for a key
func (sm *Stream) new(key string, wsClient *websocket.Conn) *Conn {
	client := &Conn{
		conn:  wsClient,
		mutex: sync.Mutex{},
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	poolIndex := sm.findPool(key)
	if poolIndex == -1 {
		// create a pool
		sm.pools = append(
			sm.pools,
			&Pool{
				Key:         key,
				connections: []*Conn{client}})
		poolIndex = len(sm.pools) - 1
		sm.Console.Log("connections["+key+"]: ", len(sm.pools[poolIndex].connections))
		return client
	}

	// use existing pool
	sm.pools[poolIndex].connections = append(
		sm.pools[poolIndex].connections,
		client)
	sm.Console.Log("connections["+key+"]: ", len(sm.pools[poolIndex].connections))
	return client
}

// Close client connection
func (sm *Stream) Close(key string, client *Conn) {
	// auxiliar clients array
	na := []*Conn{}

	// loop to remove this client
	sm.mutex.Lock()
	poolIndex := sm.findPool(key)
	for _, v := range sm.pools[poolIndex].connections {
		if v != client {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	sm.pools[poolIndex].connections = na
	sm.mutex.Unlock()
	go sm.OnUnsubscribe(key)
	client.conn.Close()
}

// Broadcast will look for pools that match a path and broadcast updates
func (sm *Stream) Broadcast(path string, opt BroadcastOpt) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	// skip pool 0 (clock)
	for poolIndex := 1; poolIndex < len(sm.pools); poolIndex++ {
		if key.Peer(sm.pools[poolIndex].Key, path) {
			data, err := opt.Get(sm.pools[poolIndex].Key)
			// this error means that the broadcast was filtered
			if err != nil {
				continue
			}

			sm.pools[poolIndex].mutex.Lock()
			modifiedData, snapshot, version := sm.Patch(poolIndex, data)
			sm.broadcast(poolIndex, opt.Encode(modifiedData), snapshot, version)
			sm.pools[poolIndex].mutex.Unlock()
			if opt.Callback != nil {
				opt.Callback()
			}
		}
	}
}

// broadcast message
func (sm *Stream) broadcast(poolIndex int, data string, snapshot bool, version int64) {
	connections := sm.pools[poolIndex].connections
	for _, client := range connections {
		sm.Write(client, data, snapshot, version)
	}
}

// Patch will return either the snapshot or the patch
//
// patch, false (patch)
//
// snapshot, true (snapshot)
func (sm *Stream) Patch(poolIndex int, data []byte) ([]byte, bool, int64) {
	patch, err := jsonpatch.CreatePatch(sm.pools[poolIndex].cache.Data, data)
	if err != nil {
		sm.Console.Err("patch create failed", err)
		version := sm._setCache(poolIndex, data)
		return data, true, version
	}
	version := sm._setCache(poolIndex, data)
	operations, err := json.Marshal(patch)
	if err != nil {
		sm.Console.Err("patch decode failed", err)
		return data, true, version
	}
	// don't send the operations if they exceed the data size
	if !sm.ForcePatch && len(operations) > len(data) {
		// sm.Console.Err("patch operations bigger than data", string(operations))
		return data, true, version
	}

	return operations, false, version
}

// Write will write data to a ws connection
func (sm *Stream) Write(client *Conn, data string, snapshot bool, version int64) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.conn.SetWriteDeadline(time.Now().Add(timeout))
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte("{"+
		"\"snapshot\": "+strconv.FormatBool(snapshot)+","+
		"\"version\": \""+strconv.FormatInt(version, 16)+"\","+
		"\"data\": \""+data+"\""+
		"}"))

	if err != nil {
		client.conn.Close()
		sm.Console.Log("writeStreamErr: ", err)
	}
}

// Read will keep alive the ws connection
func (sm *Stream) Read(key string, client *Conn) {
	for {
		_, _, err := client.conn.NextReader()
		if err != nil {
			sm.Console.Err("readSocketError["+key+"]", err)
			sm.Close(key, client)
			break
		}
	}
}

// _setCache will store data in a pool's cache
func (sm *Stream) _setCache(poolIndex int, data []byte) int64 {
	now := time.Now().UTC().UnixNano()
	sm.pools[poolIndex].cache.Version = now
	sm.pools[poolIndex].cache.Data = data
	return now
}

// SetCache by key
func (sm *Stream) setCache(key string, data []byte) int64 {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	poolIndex := sm.findPool(key)
	if poolIndex == -1 {
		now := time.Now().UTC().UnixNano()
		// create a pool
		sm.pools = append(
			sm.pools,
			&Pool{
				Key: key,
				cache: Cache{
					Version: now,
					Data:    data,
				},
				connections: []*Conn{}})
		return now
	}

	return sm._setCache(poolIndex, data)
}

// GetCache by key
func (sm *Stream) GetCacheVersion(key string) (int64, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	poolIndex := sm.findPool(key)
	if poolIndex == -1 {
		return 0, errors.New("stream pool not found")
	}
	sm.pools[poolIndex].mutex.RLock()
	defer sm.pools[poolIndex].mutex.RUnlock()
	if len(sm.pools[poolIndex].cache.Data) == 0 {
		return 0, errors.New("stream pool cache empty")
	}

	return sm.pools[poolIndex].cache.Version, nil
}

func (sm *Stream) Refresh(path string, getDataFn GetFn) Cache {
	raw, _ := getDataFn(path)
	if len(raw) == 0 {
		raw = objects.EmptyObject
	}
	cache := Cache{
		Data: raw,
	}
	cacheVersion, err := sm.GetCacheVersion(path)
	if err != nil {
		newVersion := sm.setCache(path, raw)
		cache.Version = newVersion
		return cache
	}

	cache.Version = cacheVersion
	return cache
}
