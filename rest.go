package samo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (app *Server) getStats(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		app.clock(w, r)
		return
	}
	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
		return
	}

	stats, err := app.Storage.Keys()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(stats))
}

func (app *Server) publish(w http.ResponseWriter, r *http.Request) {
	vkey := mux.Vars(r)["key"]
	count := strings.Count(vkey, "*")
	where := strings.Index(vkey, "*")
	event, err := app.messages.decode(r.Body)
	if !app.keys.IsValid(vkey) || count > 1 || (count == 1 && where != len(vkey)-1) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("samo: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	app.console.Log("publish", vkey)
	key := app.keys.Build(vkey)
	data, err := app.Filters.Write.check(key, []byte(event.Data), app.Static)
	if err != nil {
		app.console.Err("setError["+key+"]", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	index, err := app.Storage.Set(key, string(data))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	if app.Storage.Watch() == nil {
		go app.broadcast(key)
	}
	app.console.Log("publish", key)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{"+
		"\"index\": \""+index+"\""+
		"}")
	return
}

func (app *Server) read(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		app.ws(w, r)
		return
	}
	key := mux.Vars(r)["key"]
	if !app.keys.IsValid(key) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("samo: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
		return
	}

	app.console.Log("read", key)
	cache, err := app.stream.getPoolCache(key)
	if err != nil {
		raw, err := app.Storage.Get(key)
		if err != nil {
			app.console.Err(err)
		}
		filteredData, err := app.Filters.Read.check(key, raw, app.Static)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", err)
			return
		}
		app.stream.setPoolCache(key, filteredData)
		if len(filteredData) == 0 {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "%s", errors.New("samo: empty key"))
			return
		}
		cache = filteredData
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(cache))
}

func (app *Server) unpublish(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if !app.keys.IsValid(key) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("samo: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
		return
	}

	app.console.Log("unpublish", key)
	err := app.Storage.Del(key)

	if err != nil {
		app.console.Err(err.Error())
		if err.Error() == "leveldb: not found" || err.Error() == "samo: not found" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(w, "%s", err)
		return
	}

	if app.Storage.Watch() == nil {
		go app.broadcast(key)
	}

	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "unpublish "+key)
}
