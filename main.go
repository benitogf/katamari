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
	"strconv"
	"strings"
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

// SAMO : server router, Pool array, and server address
type SAMO struct {
	Router    *mux.Router
	clients   []*Pool
	storage   string
	separator string
	db        *leveldb.DB
	address   string
	console   *Console
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
	log func(v ...interface{})
	err func(v ...interface{})
}

// Stats : data structure of global keys
type Stats struct {
	Keys []string `json:"keys"`
}

var port = flag.Int("port", 8800, "ws service port")
var host = flag.String("host", "localhost", "ws service host")
var storage = flag.String("storage", "data/db", "path to the data folder")
var separator = flag.String("separator", "/", "string to use as key separator")

func validKey(key string, separator string) bool {
	// https://stackoverflow.com/a/26792316/6582356
	return !(strings.Contains(key, separator+separator) || strings.HasSuffix(key, separator) || strings.HasPrefix(key, separator))
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

func (app *SAMO) getStats(w http.ResponseWriter, r *http.Request) {
	// TODO: retry

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
	// TODO: retry
	switch mode {
	case "sa":
		data, err := app.db.Get([]byte(key), nil)
		if err == nil {
			return data
		}
		app.console.err("getDataError["+mode+"/"+key+"]", err)
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
					app.console.err("getDataError["+mode+"/"+key+"]", err)
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

func (app *SAMO) setData(mode string, key string, index string, subIndex string, data string) string {
	// TODO: retry
	now := time.Now().UTC().UnixNano()
	updated := now
	if index == "" {
		index = strconv.FormatInt(now, 16) + subIndex
	}
	if mode == "mo" {
		key += app.separator + index
	}

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
		return index
	}

	app.console.err("setDataError["+mode+"/"+key+"]", err)
	return ""
}

func (app *SAMO) delData(mode string, key string, index string) {
	// TODO: retry
	if mode == "mo" {
		key += app.separator + index
	}

	err := app.db.Delete([]byte(key), nil)
	if err == nil {
		app.console.log("deleted:", key)
		return
	}

	app.console.err("delDataError["+mode+"/"+key+"]", err)
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
	err := client.WriteMessage(1, []byte("{"+
		"\"data\": \""+data+"\""+
		"}"))
	if err != nil {
		app.console.err("sendDataError:", err)
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
			app.console.err("sendTimeError:", err)
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
		app.console.err("socketUpgradeError:", err)
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
	app.console.log("socketClientClosing[" + mode + "/" + key + "]")
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
	if len(na) > 0 {
		app.console.log("socketPoolEmpty[" + mode + "/" + key + "]")
	}
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
				_ = app.setData(mode, key, index, strconv.Itoa(clientIndex), data)
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
			app.console.err("socketKeyError:", key)
			return
		}
		client, err := app.newClient(w, r, mode, key)

		if err != nil {
			app.console.err("socketUpgradeError["+mode+"/"+key+"]", err)
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
				index := app.setData(mode, key, obj.Index, "R", obj.Data)
				app.sendData(app.findConnections(mode, key))
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, "{"+
					"\"index\": \""+index+"\""+
					"}")
				return
			}
		} else {
			err = errors.New("pathKeyError[" + key + "]: the provided key is not valid")
		}

		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "%s", err)
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
		app.console.err("socketUpgradeError["+mode+"/"+key+"]", err)
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
	app.separator = separator
	app.Router = mux.NewRouter()
	app.console = &Console{
		log: log.New(
			os.Stdout,
			color.BBlue("SAMO~[")+
				color.BPurple(time.Now().String())+
				color.BBlue("]"),
			0).Println,
		err: log.New(
			os.Stderr,
			color.BRed("SAMO~[")+
				color.BPurple(time.Now().String())+
				color.BRed("]"),
			0).Println}
	app.Router.HandleFunc("/", app.getStats)
	app.Router.HandleFunc("/sa/{key:[a-zA-Z0-9-:/]+$}", app.wss("sa"))
	app.Router.HandleFunc("/mo/{key:[a-zA-Z0-9-:/]+$}", app.wss("mo"))
	app.Router.HandleFunc("/r/sa/{key:[a-zA-Z0-9-:/]+$}", app.rPost("sa")).Methods("POST")
	app.Router.HandleFunc("/r/mo/{key:[a-zA-Z0-9-:/]+$}", app.rPost("mo")).Methods("POST")
	app.Router.HandleFunc("/r/sa/{key:[a-zA-Z0-9-:/]+$}", app.rGet("sa")).Methods("GET")
	app.Router.HandleFunc("/r/mo/{key:[a-zA-Z0-9-:/]+$}", app.rGet("mo")).Methods("GET")
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

func (app *SAMO) start() {
	var err error
	app.db, err = leveldb.OpenFile(app.storage, nil)
	if err == nil {
		log.Fatal(http.ListenAndServe(app.address, cors.Default().Handler(app.Router)))
	} else {
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	app := SAMO{}
	app.init(*host+":"+strconv.Itoa(*port), *storage, *separator)
	go app.start()
	app.console.log("glad to serve[" + app.address + "]")
	app.timer()
}
