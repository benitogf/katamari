package samo

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/benitogf/jsonpatch"

	"github.com/benitogf/coat"
	"github.com/gorilla/websocket"
)

// Subscribe : function to provide approval or denial of subscription
type Subscribe func(mode string, key string, remoteAddr string) error

// Unsubscribe : function callback on subscription closing
type Unsubscribe func(mode string, key string, remoteAddr string)

// conn extends the websocket connection with a mutex
// https://godoc.org/github.com/gorilla/websocket#hdr-Concurrency
type conn struct {
	mutex sync.Mutex
	conn  *websocket.Conn
}

// pool of mode/key filtered websocket connections
type pool struct {
	key         string
	mode        string
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

func (sm *stream) findPool(mode string, key string) int {
	poolIndex := -1
	for i := range sm.pools {
		if sm.pools[i].key == key && sm.pools[i].mode == mode {
			poolIndex = i
			break
		}
	}

	return poolIndex
}

func (sm *stream) findConnections(key string, separator string) []int {
	var res []int
	sm.mutex.RLock()
	for i := range sm.pools {
		if (sm.pools[i].key == key && sm.pools[i].mode == "sa") ||
			(sm.pools[i].mode == "mo" &&
				sm.Keys.isSub(sm.pools[i].key, key, separator)) {
			res = append(res, i)
		}
	}
	sm.mutex.RUnlock()

	return res
}

func (sm *stream) close(mode string, key string, client *conn) {
	// auxiliar clients array
	na := []*conn{}

	// loop to remove this client
	sm.mutex.Lock()
	poolIndex := sm.findPool(mode, key)
	for _, v := range sm.pools[poolIndex].connections {
		if v != client {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	sm.pools[poolIndex].connections = na
	sm.mutex.Unlock()
	go sm.Unsubscribe(mode, key, client.conn.UnderlyingConn().RemoteAddr().String())
	client.conn.Close()
}

func (sm *stream) new(mode string, key string, w http.ResponseWriter, r *http.Request) (*conn, int, error) {
	upgrader := websocket.Upgrader{
		// define the upgrade success
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Upgrade") == "websocket"
		},
		Subprotocols: []string{"bearer"},
	}

	wsClient, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		sm.console.Err("socketUpgradeError["+mode+"/"+key+"]", err)
		return nil, -1, err
	}

	err = sm.Subscribe(mode, key, wsClient.UnderlyingConn().RemoteAddr().String())
	if err != nil {
		return nil, -1, err
	}

	client, poolIndex := sm.open(mode, key, wsClient)
	return client, poolIndex, nil
}

func (sm *stream) open(mode string, key string, wsClient *websocket.Conn) (*conn, int) {
	client := &conn{
		conn:  wsClient,
		mutex: sync.Mutex{},
	}

	sm.mutex.Lock()
	poolIndex := sm.findPool(mode, key)
	if poolIndex == -1 {
		// create a pool
		sm.pools = append(
			sm.pools,
			&pool{
				key:         key,
				mode:        mode,
				connections: []*conn{client}})
		poolIndex = len(sm.pools) - 1
	} else {
		// use existing pool
		sm.pools[poolIndex].connections = append(
			sm.pools[poolIndex].connections,
			client)
	}
	sm.console.Log("connections["+mode+"/"+key+"]: ", len(sm.pools[poolIndex].connections))
	sm.mutex.Unlock()

	return client, poolIndex
}

func (sm *stream) setCache(poolIndex int, data []byte) {
	sm.mutex.Lock()
	sm.pools[poolIndex].cache = data
	sm.mutex.Unlock()
}

// patch will return either the snapshot or the patch
// patch, false (patch)
// snapshot, true (snapshot)
func (sm *stream) patch(poolIndex int, data []byte) ([]byte, bool) {
	cache := sm.pools[poolIndex].cache
	patch, err := jsonpatch.CreatePatch(cache, data)
	if err != nil {
		sm.console.Err("patch create failed", err)
		return data, true
	}
	sm.pools[poolIndex].cache = data
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
