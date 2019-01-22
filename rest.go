package samo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *Server) getStats(w http.ResponseWriter, r *http.Request) {
	if app.Audit(r) {
		stats, err := app.storage.getKeys()
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, string(stats))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
	return
}

func (app *Server) rPost(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vkey := mux.Vars(r)["key"]
		var err error
		if app.helpers.validKey(vkey, app.separator) {
			if app.Audit(r) {
				var obj Object
				decoder := json.NewDecoder(r.Body)
				defer r.Body.Close()
				err = decoder.Decode(&obj)
				if err == nil {
					if obj.Data != "" {
						now, key, index := app.helpers.makeIndexes(mode, vkey, obj.Index, "R", app.separator)
						if !app.helpers.checkArchetype(key, obj.Data, app.Archetypes) {
							app.console.err("setError["+mode+"/"+key+"]", "improper data")
							w.WriteHeader(http.StatusBadRequest)
							fmt.Fprintf(w, "%s", errors.New("SAMO: dataArchetypeError improper data"))
							return
						}
						index, err := app.storage.setData(mode, key, index, now, obj.Data)
						if err == nil {
							app.sendData(app.findConnections(mode, key))
							w.Header().Set("Content-Type", "application/json")
							fmt.Fprintf(w, "{"+
								"\"index\": \""+index+"\""+
								"}")
							return
						}

						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintf(w, "%s", err)
						return
					}

					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "%s", errors.New("SAMO: emptyDataError data is empty"))
				}

				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "%s", err)
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
	}
}

func (app *Server) rGet(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		if app.helpers.validKey(key, app.separator) {
			if app.Audit(r) {
				data := string(app.storage.getData(mode, key))
				if data != "" {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, data)
					return
				}

				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "%s", errors.New("SAMO: empty key"))
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
	}
}

func (app *Server) rDel(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if app.helpers.validKey(key, app.separator) {
		if app.Audit(r) {
			err := app.storage.delData("r", key, "")
			if err == nil {
				app.sendData(app.findConnections("sa", key))
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprintf(w, "deleted "+key)
				return
			}

			if err.Error() == "leveldb: not found" {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			fmt.Fprintf(w, "%s", err)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("SAMO: this request is not authorized"))
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
}
