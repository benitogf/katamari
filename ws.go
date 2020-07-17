package katamari

import (
	"net/http"
	"strconv"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/stream"
	"github.com/gorilla/mux"
)

func (app *Server) ws(w http.ResponseWriter, r *http.Request) {
	_key := mux.Vars(r)["key"]
	version := r.FormValue("v")

	client, err := app.Stream.New(_key, _key, w, r)
	if err != nil {
		return
	}

	entry := stream.Cache{}
	// send initial msg
	if key.Contains(app.InMemoryKeys, _key) {
		entry, err = app.MemFetch(_key, _key)
		if err != nil {
			app.console.Err("katamari: filtered route", err)
			return
		}
	} else {
		entry, err = app.Fetch(_key, _key)
		if err != nil {
			app.console.Err("katamari: filtered route", err)
			return
		}
	}

	if version != strconv.FormatInt(entry.Version, 16) {
		go app.Stream.Write(client, messages.Encode(entry.Data), true, entry.Version)
	}
	app.Stream.Read(_key, _key, client)
}
