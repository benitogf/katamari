package stream

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/benitogf/katamari/key"

	"github.com/benitogf/jsonpatch"

	"github.com/benitogf/coat"
	"github.com/gorilla/websocket"
)

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
	Key          string
	Filter       string
	cache        Cache
	connections  []*Conn
	nconnections []*Nconn
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

// FindConnections will look for pools that match a path
func (sm *Pools) FindConnections(path string) []int {
	var res []int
	sm.mutex.RLock()
	for i := range sm.Pools {
		if i != 0 && (sm.Pools[i].Key == path || key.Match(sm.Pools[i].Key, path)) {
			res = append(res, i)
		}
	}
	sm.mutex.RUnlock()

	return res
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
	upgrader := websocket.Upgrader{
		// define the upgrade success
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Upgrade") == "websocket"
		},
		Subprotocols: []string{"bearer"},
	}

	wsClient, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		sm.Console.Err("socketUpgradeError["+key+"]", err)
		return nil, err
	}

	err = sm.OnSubscribe(key)
	if err != nil {
		return nil, err
	}

	return sm.Open(key, filter, wsClient), nil
}

// Open a connection for a key
func (sm *Pools) Open(key string, filter string, wsClient *websocket.Conn) *Conn {
	client := &Conn{
		conn:  wsClient,
		mutex: sync.Mutex{},
	}

	sm.mutex.Lock()
	poolIndex := sm.findPool(key, filter)
	if poolIndex == -1 {
		// create a pool
		sm.Pools = append(
			sm.Pools,
			&Pool{
				Key:          key,
				Filter:       filter,
				connections:  []*Conn{client},
				nconnections: []*Nconn{}})
		poolIndex = len(sm.Pools) - 1
	} else {
		// use existing pool
		sm.Pools[poolIndex].connections = append(
			sm.Pools[poolIndex].connections,
			client)
	}
	sm.Console.Log("connections["+key+"]: ", len(sm.Pools[poolIndex].connections))
	sm.mutex.Unlock()

	return client
}

// Patch will return either the snapshot or the patch
//
// patch, false (patch)
//
// snapshot, true (snapshot)
func (sm *Pools) Patch(poolIndex int, data []byte) ([]byte, bool, int64) {
	cache := sm.GetCache(poolIndex)
	patch, err := jsonpatch.CreatePatch(cache.Data, data)
	if err != nil {
		sm.Console.Err("patch create failed", err)
		version := sm.SetCache(poolIndex, data)
		return data, true, version
	}
	version := sm.SetCache(poolIndex, data)
	operations, err := json.Marshal(patch)
	if err != nil {
		sm.Console.Err("patch decode failed", err)
		return data, true, version
	}
	// don't send the operations if they exceed the data size
	if !sm.ForcePatch && len(operations) > len(data) {
		return data, true, version
	}

	return operations, false, version
}

// Write will write data to a ws connection
func (sm *Pools) Write(client *Conn, data string, snapshot bool, version int64) {
	client.mutex.Lock()
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte("{"+
		"\"snapshot\": "+strconv.FormatBool(snapshot)+","+
		"\"version\": \""+strconv.FormatInt(version, 16)+"\","+
		"\"data\": \""+data+"\""+
		"}"))
	client.mutex.Unlock()
	if err != nil {
		sm.Console.Log("writeStreamErr: ", err)
	}
}

// Broadcast message
func (sm *Pools) Broadcast(poolIndex int, data string, snapshot bool, version int64) {
	sm.mutex.RLock()
	connections := sm.Pools[poolIndex].connections
	nconnections := sm.Pools[poolIndex].nconnections
	sm.mutex.RUnlock()

	for _, client := range connections {
		go sm.Write(client, data, snapshot, version)
	}
	for _, client := range nconnections {
		go sm.WriteNs(client, data, snapshot)
	}
}

// Read will keep alive the ws connection
func (sm *Pools) Read(key string, filter string, client *Conn) {
	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			sm.Console.Err("readSocketError["+key+"]", err)
			break
		}
	}
	sm.Close(key, filter, client)
}
