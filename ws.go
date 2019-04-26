package samo

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *Server) sendData(key string) {
	connections := app.stream.findConnections(key, app.separator)
	if len(connections) > 0 {
		app.stream.mutex.RLock()
		for _, poolIndex := range connections {
			key := app.stream.pools[poolIndex].key
			raw, _ := app.Storage.Get(app.stream.pools[poolIndex].mode, key)
			filteredData, err := app.Filters.Send.check(key, raw, app.Static)
			if err == nil {
				modifiedData, snapshot := app.stream.patch(poolIndex, filteredData)
				data := app.messages.encode(modifiedData)
				for _, client := range app.stream.pools[poolIndex].connections {
					go app.stream.write(client, data, snapshot)
				}
			}
		}
		app.stream.mutex.RUnlock()
	}
}

func (app *Server) processDel(mode string, key string, index string) {
	delKey := app.keys.get(mode, key, index, app.separator)
	app.console.Log("del", delKey)
	err := app.Storage.Del(delKey)
	if err != nil {
		app.console.Err("delEventError", err)
		return
	}

	app.sendData(delKey)
}

func (app *Server) processSet(mode string, key string, index string, sub string, data string) {
	setKey, setIndex, now := app.keys.Build(
		mode,
		key,
		index,
		sub,
		app.separator,
	)

	filteredData, err := app.Filters.Receive.check(setKey, []byte(data), app.Static)

	if err != nil {
		app.console.Err("setEventFiltered", err)
		return
	}

	app.console.Log("set", setKey)
	_, err = app.Storage.Set(setKey, setIndex, now, app.messages.encode(filteredData))
	if err != nil {
		app.console.Err("setEventError", err)
		return
	}

	app.sendData(setKey)
}

func (app *Server) processMessage(mode string, key string, message []byte, client *conn, r *http.Request) {
	event, err := app.messages.decode(message)
	if err != nil {
		app.console.Err("eventMessageError["+mode+"/"+key+"]", err)
		return
	}

	if !app.AuditEvent(r, event) {
		app.console.Err("socketEventUnauthorized", key)
		return
	}

	if event.Op == "del" && (event.Index != "" || mode == "sa") {
		app.processDel(mode, key, event.Index)
		return
	}

	if event.Data != "" {
		sub := fmt.Sprintf("%x", md5.Sum([]byte(client.conn.UnderlyingConn().RemoteAddr().String())))
		app.processSet(mode, key, event.Index, sub, event.Data)
	}
}

func (app *Server) readClient(mode string, key string, client *conn, r *http.Request) {
	for {
		_, message, err := client.conn.ReadMessage()

		if err != nil {
			app.console.Err("readSocketError["+mode+"/"+key+"]", err)
			break
		}

		go app.processMessage(mode, key, message, client, r)
	}
}

func (app *Server) ws(mode string) func(w http.ResponseWriter, r *http.Request) {
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

		client, poolIndex, err := app.stream.new(mode, key, w, r)

		if err != nil {
			return
		}

		// defered client close
		defer app.stream.close(mode, key, client)

		// send initial msg
		raw, _ := app.Storage.Get(mode, key)
		filteredData, err := app.Filters.Send.check(key, raw, app.Static)
		if err != nil {
			app.console.Err("samo: filtered route", err)
			return
		}

		app.stream.setCache(poolIndex, filteredData)
		go app.stream.write(client, app.messages.encode(filteredData), true)
		app.readClient(mode, key, client, r)
	}
}
