package katamari

import (
	"net/http"
	"strconv"

	"github.com/benitogf/katamari/messages"
	"github.com/gorilla/mux"
)

func (app *Server) ws(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	version := r.FormValue("v")

	client, err := app.Stream.New(key, key, w, r)
	if err != nil {
		return
	}

	// send initial msg
	entry, err := app.Fetch(key, key)
	if err != nil {
		app.console.Err("katamari: filtered route", err)
		return
	}

	if version != strconv.FormatInt(entry.Version, 16) {
		go app.Stream.Write(client, messages.Encode(entry.Data), true, entry.Version)
	}
	app.Stream.Read(key, key, client)
}
