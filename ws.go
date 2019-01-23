package samo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func (app *Server) writeToClient(client *websocket.Conn, data string) {
	err := client.WriteMessage(websocket.TextMessage, []byte("{"+
		"\"data\": \""+data+"\""+
		"}"))
	if err != nil {
		app.console.err("sendError", err)
	}
}

func (app *Server) sendData(clients []int) {
	if len(clients) > 0 {
		for _, clientIndex := range clients {
			raw, _ := app.Storage.Get(app.clients[clientIndex].mode, app.clients[clientIndex].key)
			data := app.helpers.encodeData(raw)
			for _, client := range app.clients[clientIndex].connections {
				app.writeToClient(client, data)
			}
		}
	}
}

func (app *Server) sendTime(clients []*websocket.Conn) {
	now := time.Now().UTC().UnixNano()
	data := strconv.FormatInt(now, 10)
	for _, client := range clients {
		err := client.WriteMessage(1, []byte("{"+
			"\"data\": \""+data+"\""+
			"}"))
		if err != nil {
			app.console.err("sendTimeError", err)
		}
	}
}

func (app *Server) findPool(mode string, key string) int {
	poolIndex := -1
	for i := range app.clients {
		if app.clients[i].key == key && app.clients[i].mode == mode {
			poolIndex = i
			break
		}
	}

	return poolIndex
}

func (app *Server) findConnections(key string) []int {
	var res []int
	for i := range app.clients {
		if (app.clients[i].key == key && app.clients[i].mode == "sa") ||
			(app.clients[i].mode == "mo" && app.helpers.IsMO(app.clients[i].key, key, app.separator)) {
			res = append(res, i)
		}
	}

	return res
}

func (app *Server) findClient(poolIndex int, client *websocket.Conn) int {
	clientIndex := -1
	for i := range app.clients[poolIndex].connections {
		if app.clients[poolIndex].connections[i] == client {
			clientIndex = i
			break
		}
	}

	return clientIndex
}

func (app *Server) newClient(w http.ResponseWriter, r *http.Request, mode string, key string) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		// define the upgrade success
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Upgrade") == "websocket"
		},
	}

	client, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		app.console.err("socketUpgradeError["+mode+"/"+key+"]", err)
		return nil, err
	}

	poolIndex := app.findPool(mode, key)

	if poolIndex == -1 {
		// create a pool
		app.clients = append(
			app.clients,
			&Pool{
				key:         key,
				mode:        mode,
				connections: []*websocket.Conn{client}})
		poolIndex = len(app.clients) - 1
	} else {
		// use existing pool
		app.clients[poolIndex].connections = append(
			app.clients[poolIndex].connections,
			client)
	}

	app.console.log("socketClients["+mode+"/"+key+"]", len(app.clients[poolIndex].connections))
	return client, nil
}

func (app *Server) closeClient(client *websocket.Conn, mode string, key string) {
	// remove the client before closing
	app.console.err("socketClientClosing[" + mode + "/" + key + "]")
	poolIndex := app.findPool(mode, key)

	// auxiliar clients array
	na := []*websocket.Conn{}

	// loop to remove this client
	for _, v := range app.clients[poolIndex].connections {
		if v == client {
			continue
		} else {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	app.clients[poolIndex].connections = na
	client.Close()
}

func (app *Server) readClient(client *websocket.Conn, mode string, key string) {
	for {
		_, message, err := client.ReadMessage()

		if err != nil {
			app.console.err("readSocketError["+mode+"/"+key+"]", err)
			break
		}

		var wsEvent map[string]interface{}
		err = json.Unmarshal(message, &wsEvent)
		if err != nil {
			app.console.err("jsonUnmarshalMessageError["+mode+"/"+key+"]", err)
			break
		}
		op := app.helpers.extractNonNil(wsEvent, "op")
		index := app.helpers.extractNonNil(wsEvent, "index")
		data := app.helpers.extractNonNil(wsEvent, "data")

		switch op {
		case "":
			if data != "" {
				poolIndex := app.findPool(mode, key)
				clientIndex := app.findClient(poolIndex, client)
				now, key, vindex := app.helpers.makeIndexes(mode, key, index, strconv.Itoa(clientIndex), app.separator)
				if app.helpers.checkArchetype(key, data, app.Archetypes) {
					newIndex, err := app.Storage.Set(key, vindex, now, data)
					if err == nil && newIndex != "" {
						app.sendData(app.findConnections(key))
					}
				}
			}
			break
		case "DEL":
			if index != "" || mode == "sa" {
				if mode == "mo" {
					key = key + app.separator + index
				}
				err := app.Storage.Del(key)
				if err == nil {
					app.sendData(app.findConnections(key))
				}
			}
			break
		}
	}
}

func (app *Server) wss(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		if !app.helpers.validKey(key, app.separator) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
			app.console.err("socketKeyError", key)
			return
		}
		if !app.Audit(r) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
			app.console.err("socketConnectionUnauthorized", key)
			return
		}
		client, err := app.newClient(w, r, mode, key)

		if err != nil {
			return
		}

		// send initial msg
		raw, _ := app.Storage.Get(mode, key)
		data := app.helpers.encodeData(raw)
		app.writeToClient(client, data)

		// defered client close
		defer app.closeClient(client, mode, key)

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

	for {
		_, _, err := client.ReadMessage()

		if err != nil {
			app.console.err("readSocketError["+mode+"/"+key+"]", err)
			break
		}
	}

	app.sendTime([]*websocket.Conn{client})
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
