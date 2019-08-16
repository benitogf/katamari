package samo

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (app *Server) getPatch(poolIndex int, key string) (string, bool, int64, error) {
	raw, _ := app.Storage.Get(key)
	if len(raw) == 0 {
		raw = []byte(`{ "created": 0, "updated": 0, "index": "", "data": "e30=" }`)
	}
	filteredData, err := app.Filters.Read.check(key, raw, app.Static)
	if err != nil {
		return "", false, 0, err
	}
	modifiedData, snapshot, version := app.stream.patch(poolIndex, filteredData)
	return app.messages.encode(modifiedData), snapshot, version, nil
}

func (app *Server) broadcast(key string) {
	for _, poolIndex := range app.stream.findConnections(key) {
		data, snapshot, version, err := app.getPatch(
			poolIndex,
			app.stream.pools[poolIndex].key)
		if err != nil {
			continue
		}
		go app.stream.broadcast(poolIndex, data, snapshot, version)
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
	version := r.FormValue("v")
	if !app.keys.IsValid(key) {
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
	cache, err := app.stream.getPoolCache(key)
	if err != nil {
		raw, _ := app.Storage.Get(key)
		if len(raw) == 0 {
			raw = []byte(`{ "created": 0, "updated": 0, "index": "", "data": "e30=" }`)
		}
		filteredData, err := app.Filters.Read.check(key, raw, app.Static)
		if err != nil {
			app.console.Err("samo: filtered route", err)
			return
		}
		newVersion := app.stream.setCache(poolIndex, filteredData)
		cache = vCache{
			version: newVersion,
			data:    filteredData,
		}
	}

	if version != strconv.FormatInt(cache.version, 16) {
		go app.stream.write(client, app.messages.encode(cache.data), true, cache.version)
	}
	app.readClient(key, client)
}
