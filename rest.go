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
		fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
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

		if !app.helpers.validKey(vkey, app.separator) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
			return
		}

		if !app.Audit(r) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
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
			fmt.Fprintf(w, "%s", errors.New("SAMO: emptyDataError data is empty"))
			return
		}

		now, key, index := app.helpers.makeIndexes(mode, vkey, obj.Index, "R", app.separator)

		if !app.helpers.checkArchetype(key, index, obj.Data, app.Archetypes) {
			app.console.err("setError["+mode+"/"+key+"]", "improper data")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("SAMO: dataArchetypeError improper data"))
			return
		}

		index, err = app.Storage.Set(key, index, now, obj.Data)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
			return
		}

		app.sendData(app.findConnections(key))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{"+
			"\"index\": \""+index+"\""+
			"}")
	}
}

func (app *Server) rGet(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		if !app.helpers.validKey(key, app.separator) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
			return
		}

		if !app.Audit(r) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
			return
		}

		raw, err := app.Storage.Get(mode, key)
		if err != nil {
			app.console.err(err)
		}
		data := string(raw)
		if data == "" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "%s", errors.New("SAMO: empty key"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, data)
	}
}

func (app *Server) rDel(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if !app.helpers.validKey(key, app.separator) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
		return
	}

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
		return
	}

	err := app.Storage.Del(key)

	if err != nil {
		app.console.err(err.Error())
		if err.Error() == "leveldb: not found" || err.Error() == "SAMO: not found" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(w, "%s", err)
		return
	}

	app.sendData(app.findConnections(key))
	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "deleted "+key)
}
