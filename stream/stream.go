package stream

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/benitogf/katamari/key"

	"github.com/benitogf/jsonpatch"

	"github.com/benitogf/coat"
	"github.com/gorilla/websocket"
)

const timeout = 15 * time.Second

// Subscribe : monitoring or filtering of subscriptions
type Subscribe func(key string) error

// Unsubscribe : function callback on subscription closing
type Unsubscribe func(key string)

// Conn extends the websocket connection with a mutex
// https://godoc.org/github.com/gorilla/websocket#hdr-Concurrency
type Conn struct {
	mutex sync.Mutex
	conn  *websocket.Conn
}

// Pool of key filtered connections
type Pool struct {
	Key         string
	Filter      string
	cache       Cache
	connections []*Conn
}

// Pools a group of pools
type Pools struct {
	mutex         sync.RWMutex
	OnSubscribe   Subscribe
	OnUnsubscribe Unsubscribe
	ForcePatch    bool
	Pools         []*Pool
	Console       *coat.Console
}

var StreamUpgrader = websocket.Upgrader{
	// define the upgrade success
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Upgrade") == "websocket"
	},
	Subprotocols: []string{"bearer"},
}

func (sm *Pools) findPool(key string, filter string) int {
	poolIndex := -1
	for i := range sm.Pools {
		if sm.Pools[i].Key == key && sm.Pools[i].Filter == filter {
			poolIndex = i
			break
		}
	}

	return poolIndex
}

// UseConnections will look for pools that match a path and call a function for each one
func (sm *Pools) UseConnections(path string, callback func(int)) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	for i := range sm.Pools {
		if i != 0 && (sm.Pools[i].Key == path || key.Match(sm.Pools[i].Key, path) || key.Match(path, sm.Pools[i].Key)) {
			callback(i)
		}
	}
}

// Close client connection
func (sm *Pools) Close(key string, filter string, client *Conn) {
	// auxiliar clients array
	na := []*Conn{}

	// loop to remove this client
	sm.mutex.Lock()
	poolIndex := sm.findPool(key, filter)
	for _, v := range sm.Pools[poolIndex].connections {
		if v != client {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	sm.Pools[poolIndex].connections = na
	sm.mutex.Unlock()
	go sm.OnUnsubscribe(key)
	client.conn.Close()
}

// New stream on a key
func (sm *Pools) New(key string, filter string, w http.ResponseWriter, r *http.Request) (*Conn, error) {
	wsClient, err := StreamUpgrader.Upgrade(w, r, nil)

	if err != nil {
		sm.Console.Err("socketUpgradeError["+key+"]", err)
		return nil, err
	}

	err = sm.OnSubscribe(key)
	if err != nil {
		return nil, err
	}

	return sm._New(key, filter, wsClient), nil
}

// Open a connection for a key
func (sm *Pools) _New(key string, filter string, wsClient *websocket.Conn) *Conn {
	client := &Conn{
		conn:  wsClient,
		mutex: sync.Mutex{},
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	poolIndex := sm.findPool(key, filter)
	if poolIndex == -1 {
		// create a pool
		sm.Pools = append(
			sm.Pools,
			&Pool{
				Key:         key,
				Filter:      filter,
				connections: []*Conn{client}})
		poolIndex = len(sm.Pools) - 1
		sm.Console.Log("connections["+key+"]: ", len(sm.Pools[poolIndex].connections))
		return client
	}

	// use existing pool
	sm.Pools[poolIndex].connections = append(
		sm.Pools[poolIndex].connections,
		client)
	sm.Console.Log("connections["+key+"]: ", len(sm.Pools[poolIndex].connections))
	return client
}

// Patch will return either the snapshot or the patch
//
// patch, false (patch)
//
// snapshot, true (snapshot)
func (sm *Pools) Patch(poolIndex int, data []byte) ([]byte, bool, int64) {
	cache := sm.getCache(poolIndex)
	patch, err := jsonpatch.CreatePatch(cache.Data, data)
	if err != nil {
		sm.Console.Err("patch create failed", err)
		version := sm.setCache(poolIndex, data)
		return data, true, version
	}
	version := sm.setCache(poolIndex, data)
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
func (sm *Pools) Write(client *Conn, data string, snapshot bool, version int64) {
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

// Broadcast message
func (sm *Pools) Broadcast(poolIndex int, data string, snapshot bool, version int64) {
	connections := sm.Pools[poolIndex].connections
	for _, client := range connections {
		go sm.Write(client, data, snapshot, version)
	}
}

// Read will keep alive the ws connection
func (sm *Pools) Read(key string, filter string, client *Conn) {
	for {
		_, _, err := client.conn.NextReader()
		if err != nil {
			sm.Console.Err("readSocketError["+key+"]", err)
			sm.Close(key, filter, client)
			break
		}
	}
}

// SetCache by key
func (sm *Pools) SetCache(key string, data []byte) int64 {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	poolIndex := sm.findPool(key, key)
	now := time.Now().UTC().UnixNano()
	if poolIndex == -1 {
		// create a pool
		sm.Pools = append(
			sm.Pools,
			&Pool{
				Key: key,
				cache: Cache{
					Version: now,
					Data:    data,
				},
				connections: []*Conn{}})
		return now
	}

	sm.Pools[poolIndex].cache = Cache{
		Version: now,
		Data:    data,
	}
	return now
}

// GetCache by key
func (sm *Pools) GetCache(key string) (Cache, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	poolIndex := sm.findPool(key, key)
	if poolIndex == -1 {
		return Cache{}, errors.New("stream pool not found")
	}
	cache := sm.Pools[poolIndex].cache
	if len(cache.Data) == 0 {
		return cache, errors.New("stream pool cache empty")
	}

	return cache, nil
}
