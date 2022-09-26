package katamari

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (app *Server) ws(w http.ResponseWriter, r *http.Request) {
	_key := mux.Vars(r)["key"]
	version := r.FormValue("v")

	client, err := app.Stream.New(_key, w, r)
	if err != nil {
		return
	}

	// send initial msg
	entry, err := app.fetch(_key)
	if err != nil {
		app.Console.Err("katamari: filtered route", err)
		return
	}

	if version != strconv.FormatInt(entry.Version, 16) {
		go app.Stream.Write(client, string(entry.Data), true, entry.Version)
	}
	app.Stream.Read(_key, client)
}
