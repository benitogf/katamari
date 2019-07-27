package samo

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gorilla/mux"
)

func (app *Server) getStats(w http.ResponseWriter, r *http.Request) {
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

func (app *Server) rPost(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vkey := mux.Vars(r)["key"]
		if !app.keys.isValid(vkey, app.separator) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("samo: pathKeyError key is not valid"))
			return
		}

		if !app.Audit(r) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
			return
		}

		event, err := app.messages.decodePost(r.Body)
		defer r.Body.Close()

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", err)
			return
		}

		app.console.Log("rpost", vkey)
		key, index, now := app.keys.Build(mode, vkey, event.Index, "R", app.separator)

		data, err := app.Filters.Receive.check(key, []byte(event.Data), app.Static)
		if err != nil {
			app.console.Err("setError["+mode+"/"+key+"]", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", err)
			return
		}

		index, err = app.Storage.Set(key, index, now, string(data))

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
			return
		}

		if reflect.TypeOf(app.Storage).String() != "*samo.EtcdStorage" {
			go app.sendData(key)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{"+
			"\"index\": \""+index+"\""+
			"}")
	}
}

func (app *Server) rGet(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		if !app.keys.isValid(key, app.separator) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("samo: pathKeyError key is not valid"))
			return
		}

		if !app.Audit(r) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
			return
		}

		app.console.Log("rget", key)
		raw, err := app.Storage.Get(mode, key)
		if err != nil {
			app.console.Err(err)
		}
		filteredData, err := app.Filters.Send.check(key, raw, app.Static)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", err)
		}
		data := string(filteredData)
		if data == "" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "%s", errors.New("samo: empty key"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, data)
	}
}

func (app *Server) rDel(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if !app.keys.isValid(key, app.separator) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("samo: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
		return
	}

	app.console.Log("rdel", key)
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

	if reflect.TypeOf(app.Storage).String() != "*samo.EtcdStorage" {
		go app.sendData(key)
	}
	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "deleted "+key)
}
