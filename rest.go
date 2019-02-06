package samo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

		var obj Object
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		err := decoder.Decode(&obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", err)
			return
		}

		if obj.Data == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("samo: emptyDataError data is empty"))
			return
		}

		key, index, now := app.keys.Build(mode, vkey, obj.Index, "R", app.separator)

		data, err := app.Filters.Receive.check(key, []byte(obj.Data), app.Static)
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

		go app.sendData(app.findConnections(key))
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

	go app.sendData(app.findConnections(key))
	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "deleted "+key)
}
