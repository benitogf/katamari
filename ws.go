package samo

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *Server) getPatch(poolIndex int, key string) (string, bool, error) {
	raw, _ := app.Storage.Get(key)
	if len(raw) == 0 {
		raw = []byte(`{ "created": 0, "updated": 0, "index": "", "data": "e30=" }`)
	}
	filteredData, err := app.Filters.Read.check(key, raw, app.Static)
	if err != nil {
		return "", false, err
	}
	modifiedData, snapshot := app.stream.patch(poolIndex, filteredData)
	return app.messages.encode(modifiedData), snapshot, nil
}

func (app *Server) sendData(key string) {
	for _, poolIndex := range app.stream.findConnections(key) {
		data, snapshot, err := app.getPatch(
			poolIndex,
			app.stream.pools[poolIndex].key)
		if err != nil {
			continue
		}
		go app.stream.broadcast(poolIndex, data, snapshot)
	}
}

func (app *Server) readClient(key string, client *conn) {
	for {
		_, _, err := client.conn.ReadMessage()

		if err != nil {
			app.console.Err("readSocketError["+key+"]", err)
			break
		}
	}
}

func (app *Server) ws(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if !app.keys.isValid(key) {
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

	client, poolIndex, err := app.stream.new(key, w, r)

	if err != nil {
		return
	}

	// defered client close
	defer app.stream.close(key, client)

	// send initial msg
	raw, _ := app.Storage.Get(key)
	if len(raw) == 0 {
		raw = []byte(`{ "created": 0, "updated": 0, "index": "", "data": "e30=" }`)
	}
	filteredData, err := app.Filters.Read.check(key, raw, app.Static)
	if err != nil {
		app.console.Err("samo: filtered route", err)
		return
	}
	app.stream.setCache(poolIndex, filteredData)
	go app.stream.write(client, app.messages.encode(filteredData), true)
	app.readClient(key, client)
}
