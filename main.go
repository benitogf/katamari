package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bclicn/color"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Pool : mode/key based websocket connections and watcher
type Pool struct {
	key         string
	mode        string
	connections []*websocket.Conn
}

// Archetype : function to check proper key->data covalent bond
type Archetype func(data string) bool

// Archetypes : a map that allows structure and content formalization of key->data
type Archetypes map[string]Archetype

// SAMO : server router, Pool array, and server address
type SAMO struct {
	Server     *http.Server
	Router     *mux.Router
	clients    []*Pool
	archetypes Archetypes
	storage    string
	separator  string
	db         *leveldb.DB
	address    string
	console    *Console
	closing    bool
}

// Object : data structure of elements
type Object struct {
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	Index   string `json:"index"`
	Data    string `json:"data"`
}

// Console : interface to formated terminal output
type Console struct {
	_log *log.Logger
	_err *log.Logger
	log  func(v ...interface{})
	err  func(v ...interface{})
}

// Stats : data structure of global keys
type Stats struct {
	Keys []string `json:"keys"`
}

var port = flag.Int("port", 8800, "ws service port")
var host = flag.String("host", "localhost", "ws service host")
var storage = flag.String("storage", "data/db", "path to the data folder")
var separator = flag.String("separator", "/", "character to use as key separator")

func validKey(key string, separator string) bool {
	// https://stackoverflow.com/a/26792316/6582356
	return !strings.Contains(key, separator+separator)
}

func isMO(key string, index string, separator string) bool {
	moIndex := strings.Split(strings.Replace(index, key+separator, "", 1), separator)
	return len(moIndex) == 1 && moIndex[0] != key
}

func extractNonNil(event map[string]interface{}, field string) string {
	data := ""
	if event[field] != nil {
		data = event[field].(string)
	}

	return data
}

func generateRouteRegex(separator string) string {
	return "[a-zA-Z\\d][a-zA-Z\\d\\" + separator + "]+[a-zA-Z\\d]"
}

func (app *SAMO) checkArchetype(mode string, key string, data string) bool {
	path := mode + "/" + key
	if app.archetypes[path] != nil {
		return app.archetypes[path](data)
	}

	return true
}

func (app *SAMO) checkDb() {
	tryes := 0
	if app.db == nil || app.closing {
		for (app.db == nil && tryes < 10) || app.closing {
			tryes++
			time.Sleep(800 * time.Millisecond)
		}
		if app.db == nil {
			var err error
			app.db, err = leveldb.OpenFile(app.storage, nil)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (app *SAMO) getStats(w http.ResponseWriter, r *http.Request) {
	app.checkDb()
	iter := app.db.NewIterator(nil, nil)
	stats := Stats{}
	for iter.Next() {
		stats.Keys = append(stats.Keys, string(iter.Key()))
	}
	iter.Release()
	err := iter.Error()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		respJSON, _ := json.Marshal(stats)
		fmt.Fprintf(w, string(respJSON))
		return
	}

	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "%s", err)
}

func (app *SAMO) getData(mode string, key string) []byte {
	app.checkDb()
	switch mode {
	case "sa":
		data, err := app.db.Get([]byte(key), nil)
		if err == nil {
			return data
		}
		app.console.err("getError["+mode+"/"+key+"]", err)
	case "mo":
		iter := app.db.NewIterator(util.BytesPrefix([]byte(key)), nil)
		res := []Object{}
		for iter.Next() {
			if isMO(key, string(iter.Key()), app.separator) {
				var newObject Object
				err := json.Unmarshal(iter.Value(), &newObject)
				if err == nil {
					res = append(res, newObject)
				} else {
					app.console.err("getError["+mode+"/"+key+"]", err)
				}
			}
		}
		iter.Release()
		err := iter.Error()
		if err == nil {
			data, err := json.Marshal(res)
			if err == nil {
				return data
			}
		}
	}

	return []byte("")
}

func (app *SAMO) setData(mode string, key string, index string, subIndex string, data string) (string, error) {
	now := time.Now().UTC().UnixNano()
	updated := now

	if !app.checkArchetype(mode, key, data) {
		app.console.err("setError["+mode+"/"+key+"]", "improper data")
		return "", errors.New("SAMO: dataArchtypeError improper data")
	}

	if index == "" {
		index = strconv.FormatInt(now, 16) + subIndex
	}
	if mode == "mo" {
		key += app.separator + index
	}

	app.checkDb()
	previous, err := app.db.Get([]byte(key), nil)
	created := now
	if err != nil && err.Error() == "leveldb: not found" {
		updated = 0
	}

	if err == nil {
		var oldObject Object
		err = json.Unmarshal(previous, &oldObject)
		if err == nil {
			created = oldObject.Created
			index = oldObject.Index
		}
	}

	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	})

	err = app.db.Put(
		[]byte(key),
		dataBytes.Bytes(), nil)
	if err == nil {
		app.console.log("set[" + mode + "/" + key + "]")
		return index, nil
	}

	app.console.err("setError["+mode+"/"+key+"]", err)
	return "", err
}

func (app *SAMO) delData(mode string, key string, index string) {
	if mode == "mo" {
		key += app.separator + index
	}

	app.checkDb()
	err := app.db.Delete([]byte(key), nil)
	if err == nil {
		app.console.log("delete[" + mode + "/" + key + "]")
		return
	}

	app.console.err("deleteError["+mode+"/"+key+"]", err)
	return
}

func (app *SAMO) getEncodedData(mode string, key string) string {
	raw := app.getData(mode, key)
	data := ""
	if len(raw) > 0 {
		data = base64.StdEncoding.EncodeToString(raw)
	}

	return data
}

func (app *SAMO) writeToClient(client *websocket.Conn, data string) {
	err := client.WriteMessage(websocket.TextMessage, []byte("{"+
		"\"data\": \""+data+"\""+
		"}"))
	if err != nil {
		app.console.err("sendError", err)
	}
}

func (app *SAMO) sendData(clients []int) {
	if len(clients) > 0 {
		for _, clientIndex := range clients {
			data := app.getEncodedData(app.clients[clientIndex].mode, app.clients[clientIndex].key)
			for _, client := range app.clients[clientIndex].connections {
				app.writeToClient(client, data)
			}
		}
	}
}

func (app *SAMO) sendTime(clients []*websocket.Conn) {
	now := time.Now().UTC().UnixNano()
	data := strconv.FormatInt(now, 10)
	for _, client := range clients {
		err := client.WriteMessage(1, []byte("{"+
			"\"data\": \""+data+"\""+
			"}"))
		if err != nil {
			app.console.err("sendTimeError", err)
		}
	}
}

func (app *SAMO) findPool(mode string, key string) int {
	poolIndex := -1
	for i := range app.clients {
		if app.clients[i].key == key && app.clients[i].mode == mode {
			poolIndex = i
			break
		}
	}

	return poolIndex
}

func (app *SAMO) findConnections(mode string, key string) []int {
	var res []int
	for i := range app.clients {
		if (app.clients[i].key == key && app.clients[i].mode == mode) ||
			(mode == "sa" && app.clients[i].mode == "mo" && isMO(app.clients[i].key, key, app.separator)) ||
			(mode == "mo" && app.clients[i].mode == "sa" && isMO(key, app.clients[i].key, app.separator)) {
			res = append(res, i)
		}
	}

	return res
}

func (app *SAMO) findClient(poolIndex int, client *websocket.Conn) int {
	clientIndex := -1
	for i := range app.clients[poolIndex].connections {
		if app.clients[poolIndex].connections[i] == client {
			clientIndex = i
			break
		}
	}

	return clientIndex
}

func (app *SAMO) newClient(w http.ResponseWriter, r *http.Request, mode string, key string) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		// define the upgrade success
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Upgrade") == "websocket"
		},
	}

	client, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		app.console.err("socketUpgradeError["+mode+"/"+key+"]", err)
		return nil, err
	}

	poolIndex := app.findPool(mode, key)

	if poolIndex == -1 {
		// create a pool
		app.clients = append(
			app.clients,
			&Pool{
				key:         key,
				mode:        mode,
				connections: []*websocket.Conn{client}})
		poolIndex = len(app.clients) - 1
	} else {
		// use existing pool
		app.clients[poolIndex].connections = append(
			app.clients[poolIndex].connections,
			client)
	}

	app.console.log("socketClients["+mode+"/"+key+"]", len(app.clients[poolIndex].connections))
	return client, nil
}

func (app *SAMO) closeClient(client *websocket.Conn, mode string, key string) {
	// remove the client before closing
	app.console.err("socketClientClosing[" + mode + "/" + key + "]")
	poolIndex := app.findPool(mode, key)

	// auxiliar clients array
	na := []*websocket.Conn{}

	// loop to remove this client
	for _, v := range app.clients[poolIndex].connections {
		if v == client {
			continue
		} else {
			na = append(na, v)
		}
	}

	// replace clients array with the auxiliar
	app.clients[poolIndex].connections = na
	client.Close()
}

func (app *SAMO) readClient(client *websocket.Conn, mode string, key string) {
	for {
		_, message, err := client.ReadMessage()

		if err != nil {
			app.console.err("readSocketError["+mode+"/"+key+"]", err)
			break
		}

		var wsEvent map[string]interface{}
		err = json.Unmarshal(message, &wsEvent)
		if err != nil {
			app.console.err("jsonUnmarshalError["+mode+"/"+key+"]", err)
			break
		}
		op := extractNonNil(wsEvent, "op")
		index := extractNonNil(wsEvent, "index")
		data := extractNonNil(wsEvent, "data")

		switch op {
		case "":
			if data != "" {
				poolIndex := app.findPool(mode, key)
				clientIndex := app.findClient(poolIndex, client)
				_, _ = app.setData(mode, key, index, strconv.Itoa(clientIndex), data)
			}
			break
		case "DEL":
			if index != "" || op == "sa" {
				app.delData(mode, key, index)
			}
			break
		}
		app.sendData(app.findConnections(mode, key))
	}
}

func (app *SAMO) wss(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		if !validKey(key, app.separator) {
			app.console.err("socketKeyError", key)
			return
		}
		client, err := app.newClient(w, r, mode, key)

		if err != nil {
			return
		}

		// send initial msg
		data := app.getEncodedData(mode, key)
		app.writeToClient(client, data)

		// defered client close
		defer app.closeClient(client, mode, key)

		app.readClient(client, mode, key)
	}
}

func (app *SAMO) rPost(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		var err error
		if validKey(key, app.separator) {
			var obj Object
			decoder := json.NewDecoder(r.Body)
			defer r.Body.Close()
			err = decoder.Decode(&obj)
			if err == nil {
				if obj.Data != "" {
					index, err := app.setData(mode, key, obj.Index, "R", obj.Data)
					if err == nil {
						app.sendData(app.findConnections(mode, key))
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprintf(w, "{"+
							"\"index\": \""+index+"\""+
							"}")
						return
					}

					w.WriteHeader(http.StatusExpectationFailed)
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

		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "%s", errors.New("SAMO: pathKeyError key is not valid"))
	}
}

func (app *SAMO) rGet(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		data := string(app.getData(mode, mux.Vars(r)["key"]))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, data)
	}
}

func (app *SAMO) timeWs(w http.ResponseWriter, r *http.Request) {
	mode := "ws"
	key := "time"
	client, err := app.newClient(w, r, mode, key)

	if err != nil {
		return
	}

	defer app.closeClient(client, mode, key)

	for {
		_, _, err := client.ReadMessage()

		if err != nil {
			app.console.err("readSocketError["+mode+"/"+key+"]", err)
			break
		}
	}

	app.sendTime([]*websocket.Conn{client})
}

func (app *SAMO) init(address string, storage string, separator string) {
	app.address = address
	app.storage = storage
	if len(separator) > 1 {
		log.Fatal("the separator needs to be one character, currently it has ", len(separator), " please change or remove this flag, the default value is \"/\"")
	}
	app.separator = separator
	app.Router = mux.NewRouter()
	app.closing = false
	app.console = &Console{
		_err: log.New(os.Stderr, "", 0),
		_log: log.New(os.Stdout, "", 0),
		log: func(v ...interface{}) {
			app.console._log.SetPrefix(color.BBlue("["+storage+"]~[") +
				color.BPurple(time.Now().Format("2006-01-02 15:04:05.000000")) +
				color.BBlue("]~"))
			app.console._log.Println(v)
		},
		err: func(v ...interface{}) {
			app.console._err.SetPrefix(color.BRed("["+storage+"]~[") +
				color.BPurple(time.Now().Format("2006-01-02 15:04:05.000000")) +
				color.BRed("]~"))
			app.console._err.Println(v)
		}}
	rr := generateRouteRegex(app.separator)
	app.Router.HandleFunc("/", app.getStats)
	app.Router.HandleFunc("/sa/{key:"+rr+"}", app.wss("sa"))
	app.Router.HandleFunc("/mo/{key:"+rr+"}", app.wss("mo"))
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rPost("sa")).Methods("POST")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rPost("mo")).Methods("POST")
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rGet("sa")).Methods("GET")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rGet("mo")).Methods("GET")
	app.Router.HandleFunc("/time", app.timeWs)
}

func (app *SAMO) timer() {
	ticker := time.NewTicker(time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			poolIndex := app.findPool("ws", "time")
			if poolIndex != -1 {
				app.sendTime(app.clients[poolIndex].connections)
			}
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func (app *SAMO) waitServer() {
	tryes := 0
	for app.Server == nil && tryes < 100 {
		tryes++
		time.Sleep(100 * time.Millisecond)
	}
	if app.Server == nil {
		log.Fatal("Server start failed")
	}
}

func (app *SAMO) start() {
	go func() {
		var err error
		app.console.log("starting db")
		app.db, err = leveldb.OpenFile(app.storage, nil)
		if err == nil {
			app.console.log("starting server")
			app.Server = &http.Server{
				Addr:    app.address,
				Handler: cors.Default().Handler(app.Router)}
			err = app.Server.ListenAndServe()
			if !app.closing {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}()
	app.waitServer()
}

func (app *SAMO) close(sig os.Signal) {
	if !app.closing {
		app.closing = true
		if app.db != nil {
			app.db.Close()
		}
		app.console.err("shutdown", sig)
		if app.Server != nil {
			app.Server.Shutdown(nil)
		}
	}
}

func (app *SAMO) waitSignal() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		app.close(sig)
		done <- true
	}()
	<-done
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	app := SAMO{}
	app.init(*host+":"+strconv.Itoa(*port), *storage, *separator)
	app.start()
	app.console.log("glad to serve[" + app.address + "]")
	go app.timer()
	app.waitSignal()
}
