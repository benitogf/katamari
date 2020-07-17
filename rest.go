package katamari

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/benitogf/katamari/stream"
	"github.com/gorilla/mux"
)

func (app *Server) getStats(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		app.clock(w, r)
		return
	}
	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("katamari: this request is not authorized"))
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
	event, err := messages.Decode(r.Body)
	if !key.IsValid(vkey) || count > 1 || (count == 1 && where != len(vkey)-1) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("katamari: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("katamari: this request is not authorized"))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	_key := key.Build(vkey)
	data, err := app.filters.Write.check(_key, []byte(event.Data), app.Static)
	if err != nil {
		app.console.Err("setError["+_key+"]", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	index := ""
	if key.Contains(app.InMemoryKeys, _key) {
		index, err = app.Storage.MemSet(_key, string(data))
	} else {
		index, err = app.Storage.Set(_key, string(data))
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	app.console.Log("publish", _key)
	app.filters.After.check(_key)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{"+
		"\"index\": \""+index+"\""+
		"}")
	return
}

func (app *Server) read(w http.ResponseWriter, r *http.Request) {
	_key := mux.Vars(r)["key"]
	if !key.IsValid(_key) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("katamari: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("katamari: this request is not authorized"))
		return
	}

	if r.Header.Get("Upgrade") == "websocket" {
		app.ws(w, r)
		return
	}

	app.console.Log("read", _key)
	entry := stream.Cache{}
	var err error
	if key.Contains(app.InMemoryKeys, _key) {
		entry, err = app.MemForceFetch(_key, _key)
	} else {
		entry, err = app.ForceFetch(_key, _key)
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}
	if bytes.Equal(entry.Data, objects.EmptyObject) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "%s", errors.New("katamari: empty key"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(entry.Data))
}

func (app *Server) unpublish(w http.ResponseWriter, r *http.Request) {
	_key := mux.Vars(r)["key"]
	if !key.IsValid(_key) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("katamari: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("katamari: this request is not authorized"))
		return
	}

	err := app.filters.Delete.check(_key, app.Static)
	if err != nil {
		app.console.Err("detError["+_key+"]", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	app.console.Log("unpublish", _key)

	if key.Contains(app.InMemoryKeys, _key) {
		err = app.Storage.MemDel(_key)
	} else {
		err = app.Storage.Del(_key)
	}

	if err != nil {
		app.console.Err(err.Error())
		if err.Error() == "leveldb: not found" || err.Error() == "katamari: not found" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(w, "%s", err)
		return
	}

	// this performs better than the watch channel
	// if app.Storage.Watch() == nil {
	// 	go app.broadcast(_key)
	// }

	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "unpublish "+_key)
}
