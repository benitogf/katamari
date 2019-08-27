package katamari

import (
	"net/http"
	"strconv"

	"github.com/benitogf/katamari/stream"
	"github.com/gorilla/mux"
)

func (app *Server) ws(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	version := r.FormValue("v")

	client, poolIndex, err := app.stream.New(key, w, r)

	if err != nil {
		return
	}

	// send initial msg
	cache, err := app.stream.GetPoolCache(key)
	if err != nil {
		raw, _ := app.Storage.Get(key)
		if len(raw) == 0 {
			raw = emptyObject
		}
		filteredData, err := app.filters.Read.check(key, raw, app.Static)
		if err != nil {
			app.console.Err("katamari: siltered route", err)
			return
		}
		newVersion := app.stream.SetCache(poolIndex, filteredData)
		cache = stream.Cache{
			Version: newVersion,
			Data:    filteredData,
		}
	}

	if version != strconv.FormatInt(cache.Version, 16) {
		go app.stream.Write(client, app.messages.Encode(cache.Data), true, cache.Version)
	}
	app.stream.Read(key, client)
}
