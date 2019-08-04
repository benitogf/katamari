package samo

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/benitogf/jsonpatch"

	"github.com/benitogf/coat"
	"github.com/gorilla/websocket"
)

// Subscribe : function to monitoring or filtering of subscription
type Subscribe func(key string) error

// Unsubscribe : function callback on subscription closing
type Unsubscribe func(key string)

// conn extends the websocket connection with a mutex
// https://godoc.org/github.com/gorilla/websocket#hdr-Concurrency
type conn struct {
	mutex sync.Mutex
	conn  *websocket.Conn
}

// pool of key filtered websocket connections
type pool struct {
	key         string
	cache       []byte
	connections []*conn
}

// stream a group of pools
type stream struct {
	mutex       sync.RWMutex
	Subscribe   Subscribe
	Unsubscribe Unsubscribe
	pools       []*pool
	console     *coat.Console
	*Keys
	*Messages
}

func (sm *stream) findPool(key string) int {
	poolIndex := -1
	for i := range sm.pools {
		if sm.pools[i].key == key {
			poolIndex = i
			break
		}
	}

	return poolIndex
}

func (sm *stream) findConnections(key string) []int {
	var res []int
	sm.mutex.RLock()
	for i := range sm.pools {
		isGlob := strings.Contains(key, "*")
		if (!isGlob && sm.pools[i].key == key) || sm.Keys.isSub(sm.pools[i].key, key) {
			res = append(res, i)
		}
	}
	sm.mutex.RUnlock()

	return res
}

func (sm *stream) close(key string, client *conn) {
	// auxiliar clients array
	na := []*conn{}

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
	go sm.Unsubscribe(key)
	client.conn.Close()
}

func (sm *stream) new(key string, w http.ResponseWriter, r *http.Request) (*conn, int, error) {
	upgrader := websocket.Upgrader{
		// define the upgrade success
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Upgrade") == "websocket"
		},
		Subprotocols: []string{"bearer"},
	}

	wsClient, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		sm.console.Err("socketUpgradeError["+key+"]", err)
		return nil, -1, err
	}

	err = sm.Subscribe(key)
	if err != nil {
		return nil, -1, err
	}

	client, poolIndex := sm.open(key, wsClient)
	return client, poolIndex, nil
}

func (sm *stream) open(key string, wsClient *websocket.Conn) (*conn, int) {
	client := &conn{
		conn:  wsClient,
		mutex: sync.Mutex{},
	}

	sm.mutex.Lock()
	poolIndex := sm.findPool(key)
	if poolIndex == -1 {
		// create a pool
		sm.pools = append(
			sm.pools,
			&pool{
				key:         key,
				connections: []*conn{client}})
		poolIndex = len(sm.pools) - 1
	} else {
		// use existing pool
		sm.pools[poolIndex].connections = append(
			sm.pools[poolIndex].connections,
			client)
	}
	sm.console.Log("connections["+key+"]: ", len(sm.pools[poolIndex].connections))
	sm.mutex.Unlock()

	return client, poolIndex
}

func (sm *stream) setCache(poolIndex int, data []byte) {
	sm.mutex.Lock()
	sm.pools[poolIndex].cache = data
	sm.mutex.Unlock()
}

func (sm *stream) getCache(poolIndex int) []byte {
	sm.mutex.RLock()
	cache := sm.pools[poolIndex].cache
	sm.mutex.RUnlock()
	return cache
}

// patch will return either the snapshot or the patch
// patch, false (patch)
// snapshot, true (snapshot)
func (sm *stream) patch(poolIndex int, data []byte) ([]byte, bool) {
	cache := sm.getCache(poolIndex)
	patch, err := jsonpatch.CreatePatch(cache, data)
	if err != nil {
		sm.console.Err("patch create failed", err)
		sm.setCache(poolIndex, data)
		return data, true
	}
	sm.setCache(poolIndex, data)
	operations, err := json.Marshal(patch)
	if err != nil {
		sm.console.Err("patch decode failed", err)
		return data, true
	}

	return operations, false
}

func (sm *stream) write(client *conn, data string, snapshot bool) {
	client.mutex.Lock()
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte("{"+
		"\"snapshot\": "+strconv.FormatBool(snapshot)+","+
		"\"data\": \""+data+"\""+
		"}"))
	client.mutex.Unlock()
	if err != nil {
		sm.console.Log("writeStreamErr: ", err)
	}
}

func (sm *stream) broadcast(poolIndex int, data string, snapshot bool) {
	sm.mutex.RLock()
	connections := sm.pools[poolIndex].connections
	sm.mutex.RUnlock()

	for _, client := range connections {
		go sm.write(client, data, snapshot)
	}
}

func (sm *stream) writeTime(client *conn, data string) {
	client.mutex.Lock()
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte(data))
	client.mutex.Unlock()
	if err != nil {
		sm.console.Log("writeStreamErr: ", err)
	}
}

func (sm *stream) broadcastTime(data string) {
	sm.mutex.RLock()
	connections := sm.pools[0].connections
	sm.mutex.RUnlock()

	for _, client := range connections {
		go sm.writeTime(client, data)
	}
}
