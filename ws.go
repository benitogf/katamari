package katamari

import (
	"net/http"
	"strconv"

	"github.com/benitogf/katamari/messages"
	"github.com/gorilla/mux"
)

func (app *Server) ws(w http.ResponseWriter, r *http.Request) error {
	_key := mux.Vars(r)["key"]
	version := r.FormValue("v")

	err := app.filters.Read.checkStatic(_key, app.Static)
	if err != nil {
		app.Console.Err("katamari: filtered route", err)
		return err
	}

	client, err := app.Stream.New(_key, w, r)
	if err != nil {
		return err
	}

	// send initial msg
	entry, err := app.fetch(_key)
	if err != nil {
		app.Console.Err("katamari: filtered route", err)
		return err
	}

	if version != strconv.FormatInt(entry.Version, 16) {
		go app.Stream.Write(client, messages.Encode(entry.Data), true, entry.Version)
	}
	app.Stream.Read(_key, client)
	return nil
}
