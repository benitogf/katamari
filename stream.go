package samo

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/benitogf/nsocket"

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

type nconn struct {
	mutex sync.Mutex
	conn  *nsocket.Client
}

type vCache struct {
	version int64
	data    []byte
}

// pool of key filtered websocket connections
type pool struct {
	key          string
	cache        vCache
	connections  []*conn
	nconnections []*nconn
}

// stream a group of pools
type stream struct {
	mutex         sync.RWMutex
	OnSubscribe   Subscribe
	OnUnsubscribe Unsubscribe
	forcePatch    bool
	pools         []*pool
	console       *coat.Console
	*Keys
	*messages
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
		if i != 0 && (sm.pools[i].key == key || sm.Keys.Match(sm.pools[i].key, key)) {
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
	go sm.OnUnsubscribe(key)
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

	err = sm.OnSubscribe(key)
	if err != nil {
		return nil, -1, err
	}

	client, poolIndex := sm.open(key, wsClient)
	return client, poolIndex, nil
}

func (sm *stream) closeNs(client *nconn) {
	// auxiliar clients array
	na := []*nconn{}

	// loop to remove this client
	sm.mutex.Lock()
	poolIndex := sm.findPool(client.conn.Path)
	for _, v := range sm.pools[poolIndex].nconnections {
		if v != client {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	sm.pools[poolIndex].nconnections = na
	sm.mutex.Unlock()
	go sm.OnUnsubscribe(client.conn.Path)
	client.conn.Close()
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
				key:          key,
				connections:  []*conn{client},
				nconnections: []*nconn{}})
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

func (sm *stream) openNs(nsClient *nsocket.Client) (*nconn, int) {
	client := &nconn{
		conn:  nsClient,
		mutex: sync.Mutex{},
	}

	sm.mutex.Lock()
	poolIndex := sm.findPool(client.conn.Path)
	if poolIndex == -1 {
		// create a pool
		sm.pools = append(
			sm.pools,
			&pool{
				key:          client.conn.Path,
				connections:  []*conn{},
				nconnections: []*nconn{client}})
		poolIndex = len(sm.pools) - 1
	} else {
		// use existing pool
		sm.pools[poolIndex].nconnections = append(
			sm.pools[poolIndex].nconnections,
			client)
	}
	sm.console.Log("nconnections["+client.conn.Path+"]: ", len(sm.pools[poolIndex].nconnections))
	sm.mutex.Unlock()

	return client, poolIndex
}

func (sm *stream) setCache(poolIndex int, data []byte) int64 {
	sm.mutex.Lock()
	now := time.Now().UTC().UnixNano()
	sm.pools[poolIndex].cache = vCache{
		version: now,
		data:    data,
	}
	sm.mutex.Unlock()
	return now
}

func (sm *stream) getCache(poolIndex int) vCache {
	sm.mutex.RLock()
	cache := sm.pools[poolIndex].cache
	sm.mutex.RUnlock()
	return cache
}

func (sm *stream) setPoolCache(key string, data []byte) int64 {
	sm.mutex.Lock()
	poolIndex := sm.findPool(key)
	now := time.Now().UTC().UnixNano()
	if poolIndex == -1 {
		// create a pool
		sm.pools = append(
			sm.pools,
			&pool{
				key: key,
				cache: vCache{
					version: now,
					data:    data,
				},
				connections: []*conn{}})
		sm.mutex.Unlock()
		return now
	}
	sm.pools[poolIndex].cache = vCache{
		version: now,
		data:    data,
	}
	sm.mutex.Unlock()

	return now
}

func (sm *stream) getPoolCache(key string) (vCache, error) {
	sm.mutex.RLock()
	poolIndex := sm.findPool(key)
	if poolIndex == -1 {
		sm.mutex.RUnlock()
		return vCache{}, errors.New("stream pool not found")
	}
	cache := sm.pools[poolIndex].cache
	if len(cache.data) == 0 {
		sm.mutex.RUnlock()
		return cache, errors.New("stream pool cache empty")
	}
	sm.mutex.RUnlock()
	return cache, nil
}

// patch will return either the snapshot or the patch
// patch, false (patch)
// snapshot, true (snapshot)
func (sm *stream) patch(poolIndex int, data []byte) ([]byte, bool, int64) {
	cache := sm.getCache(poolIndex)
	patch, err := jsonpatch.CreatePatch(cache.data, data)
	if err != nil {
		sm.console.Err("patch create failed", err)
		version := sm.setCache(poolIndex, data)
		return data, true, version
	}
	version := sm.setCache(poolIndex, data)
	operations, err := json.Marshal(patch)
	if err != nil {
		sm.console.Err("patch decode failed", err)
		return data, true, version
	}
	// don't send the operations if they exceed the data size
	if !sm.forcePatch && len(operations) > len(data) {
		return data, true, version
	}

	return operations, false, version
}

func (sm *stream) write(client *conn, data string, snapshot bool, version int64) {
	client.mutex.Lock()
	err := client.conn.WriteMessage(websocket.BinaryMessage, []byte("{"+
		"\"snapshot\": "+strconv.FormatBool(snapshot)+","+
		"\"version\": \""+strconv.FormatInt(version, 16)+"\","+
		"\"data\": \""+data+"\""+
		"}"))
	client.mutex.Unlock()
	if err != nil {
		sm.console.Log("writeStreamErr: ", err)
	}
}

func (sm *stream) writeNs(client *nconn, data string, snapshot bool) {
	client.mutex.Lock()
	err := client.conn.Write("{" +
		"\"snapshot\": " + strconv.FormatBool(snapshot) + "," +
		"\"data\": \"" + data + "\"" +
		"}")
	client.mutex.Unlock()
	if err != nil {
		sm.console.Log("writeStreamErr: ", err)
	}
}

func (sm *stream) broadcast(poolIndex int, data string, snapshot bool, version int64) {
	sm.mutex.RLock()
	connections := sm.pools[poolIndex].connections
	nconnections := sm.pools[poolIndex].nconnections
	sm.mutex.RUnlock()

	for _, client := range connections {
		go sm.write(client, data, snapshot, version)
	}
	for _, client := range nconnections {
		go sm.writeNs(client, data, snapshot)
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
