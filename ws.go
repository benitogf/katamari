package samo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func (app *Server) writeToClient(client *conn, data string) {
	client.mutex.Lock()
	err := client.conn.WriteMessage(websocket.TextMessage, []byte("{"+
		"\"data\": \""+data+"\""+
		"}"))
	client.mutex.Unlock()
	if err != nil {
		app.console.Err("sendError", err)
	}
}

func (app *Server) sendData(clients []int) {
	if len(clients) > 0 {
		for _, clientIndex := range clients {
			key := app.clients[clientIndex].key
			raw, _ := app.Storage.Get(app.clients[clientIndex].mode, key)
			filteredData, err := app.Filters.Send.check(key, raw, app.Static)
			if err == nil {
				data := app.messages.write(filteredData)
				for _, client := range app.clients[clientIndex].connections {
					go app.writeToClient(client, data)
				}
			}
		}
	}
}

func (app *Server) sendTime(clients []*conn) {
	now := time.Now().UTC().UnixNano()
	data := strconv.FormatInt(now, 10)
	for _, client := range clients {
		client.mutex.Lock()
		err := client.conn.WriteMessage(1, []byte("{"+
			"\"data\": \""+data+"\""+
			"}"))
		client.mutex.Unlock()
		if err != nil {
			app.console.Err("sendTimeError", err)
		}
	}
}

func (app *Server) findPool(mode string, key string) int {
	poolIndex := -1
	app.mutexClients.RLock()
	for i := range app.clients {
		if app.clients[i].key == key && app.clients[i].mode == mode {
			poolIndex = i
			break
		}
	}
	app.mutexClients.RUnlock()

	return poolIndex
}

func (app *Server) findConnections(key string) []int {
	var res []int
	app.mutexClients.RLock()
	for i := range app.clients {
		if (app.clients[i].key == key && app.clients[i].mode == "sa") ||
			(app.clients[i].mode == "mo" &&
				app.keys.isSub(app.clients[i].key, key, app.separator)) {
			res = append(res, i)
		}
	}
	app.mutexClients.RUnlock()

	return res
}

func (app *Server) findClient(poolIndex int, client *conn) int {
	clientIndex := -1
	app.mutexClients.RLock()
	for i := range app.clients[poolIndex].connections {
		if app.clients[poolIndex].connections[i] == client {
			clientIndex = i
			break
		}
	}
	app.mutexClients.RUnlock()

	return clientIndex
}

func (app *Server) newClient(w http.ResponseWriter, r *http.Request, mode string, key string) (*conn, error) {
	upgrader := websocket.Upgrader{
		// define the upgrade success
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Upgrade") == "websocket"
		},
	}

	wsClient, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		app.console.Err("socketUpgradeError["+mode+"/"+key+"]", err)
		return nil, err
	}

	poolIndex := app.findPool(mode, key)
	client := &conn{
		conn:  wsClient,
		mutex: sync.Mutex{},
	}

	app.mutexClients.Lock()
	if poolIndex == -1 {
		// create a pool
		app.clients = append(
			app.clients,
			&pool{
				key:         key,
				mode:        mode,
				connections: []*conn{client}})
		poolIndex = len(app.clients) - 1
	} else {
		// use existing pool
		app.clients[poolIndex].connections = append(
			app.clients[poolIndex].connections,
			client)
	}

	app.console.Log("socketClients["+mode+"/"+key+"]", len(app.clients[poolIndex].connections))
	app.mutexClients.Unlock()
	return client, nil
}

func (app *Server) closeClient(client *conn, mode string, key string) {
	// remove the client before closing
	poolIndex := app.findPool(mode, key)

	// auxiliar clients array
	na := []*conn{}

	// loop to remove this client
	app.mutexClients.Lock()
	for _, v := range app.clients[poolIndex].connections {
		if v == client {
			continue
		} else {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	app.clients[poolIndex].connections = na
	app.mutexClients.Unlock()
	client.conn.Close()
}

func (app *Server) processDel(mode string, key string, index string) {
	delKey := app.keys.get(mode, key, index, app.separator)
	app.console.Log("del", delKey)
	err := app.Storage.Del(delKey)
	if err == nil {
		go app.sendData(app.findConnections(delKey))
	}
}

func (app *Server) processSet(mode string, key string, index string, data string, client *conn) {
	poolIndex := app.findPool(mode, key)
	clientIndex := strconv.Itoa(app.findClient(poolIndex, client))
	setKey, setIndex, now := app.keys.build(
		mode,
		key,
		index,
		clientIndex,
		app.separator,
	)

	filteredData, err := app.Filters.Receive.check(setKey, []byte(data), app.Static)
	if err == nil {
		app.console.Log("set", setKey)
		newIndex, err := app.Storage.Set(setKey, setIndex, now, string(filteredData))
		if err == nil && newIndex != "" {
			go app.sendData(app.findConnections(setKey))
		}
	}
}

func (app *Server) processMessage(mode string, key string, message []byte, client *conn) {
	var wsEvent map[string]interface{}
	err := json.Unmarshal(message, &wsEvent)
	if err != nil {
		app.console.Err("jsonUnmarshalMessageError["+mode+"/"+key+"]", err)
		return
	}
	op := app.messages.extract(wsEvent, "op")
	index := app.messages.extract(wsEvent, "index")
	data := app.messages.extract(wsEvent, "data")

	if op == "del" && (index != "" || mode == "sa") {
		go app.processDel(mode, key, index)
		return
	}

	if data != "" {
		go app.processSet(mode, key, index, data, client)
	}
}

func (app *Server) readClient(client *conn, mode string, key string) {
	for {
		_, message, err := client.conn.ReadMessage()

		if err != nil {
			app.console.Err("readSocketError["+mode+"/"+key+"]", err)
			break
		}

		go app.processMessage(mode, key, message, client)
	}
}

func (app *Server) wss(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		if !app.keys.isValid(key, app.separator) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("samo: pathKeyError key is not valid"))
			app.console.Err("socketKeyError", key)
			return
		}
		if !app.Audit(r) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
			app.console.Err("socketConnectionUnauthorized", key)
			return
		}
		client, err := app.newClient(w, r, mode, key)

		if err != nil {
			return
		}

		// defered client close
		defer app.closeClient(client, mode, key)

		// send initial msg
		raw, _ := app.Storage.Get(mode, key)
		filteredData, err := app.Filters.Send.check(key, raw, app.Static)
		if err == nil {
			go app.writeToClient(client, app.messages.write(filteredData))
		}

		app.readClient(client, mode, key)
	}
}

func (app *Server) timeWs(w http.ResponseWriter, r *http.Request) {
	mode := "ws"
	key := "time"
	client, err := app.newClient(w, r, mode, key)

	if err != nil {
		return
	}

	defer app.closeClient(client, mode, key)
	app.sendTime([]*conn{client})

	for {
		_, _, err := client.conn.ReadMessage()

		if err != nil {
			app.console.Err("readSocketError["+mode+"/"+key+"]", err)
			break
		}
	}
}

func (app *Server) timer() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			poolIndex := app.findPool("ws", "time")
			if poolIndex != -1 {
				app.sendTime(app.clients[poolIndex].connections)
			}
		}
	}
}
