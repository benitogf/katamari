package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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

// App : server router, Pool array, and server address
type App struct {
	Router  *mux.Router
	clients []*Pool
	db      *leveldb.DB
	address string
}

// Object : data structure of elements
type Object struct {
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	Index   string `json:"index"`
	Data    string `json:"data"`
}

// Stats : data structure of global keys
type Stats struct {
	Keys []string `json:"keys"`
}

var port = flag.Int("port", 8800, "ws service port")
var host = flag.String("host", "localhost", "ws service host")
var storage = flag.String("storage", "data/db", "path to the data folder")
var separator = "/"

var stdout *log.Logger
var stderr *log.Logger

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (app *App) getStats(w http.ResponseWriter, r *http.Request) {
	// TODO: retry

	iter := app.db.NewIterator(nil, nil)
	stats := Stats{}
	for iter.Next() {
		key := string(iter.Key())
		stats.Keys = append(stats.Keys, key)
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

func (app *App) getData(mode string, key string) []byte {
	// TODO: retry
	switch mode {
	case "sa":
		data, err := app.db.Get([]byte(key), nil)
		if err == nil {
			var newObject Object
			err = json.Unmarshal(data, &newObject)
			if err == nil {
				data, err := json.Marshal(newObject)
				if err == nil {
					return data
				}
			}
		}
		stderr.Println("getDataError:", err)
	case "mo":
		iter := app.db.NewIterator(util.BytesPrefix([]byte(key)), nil)
		res := []Object{}
		for iter.Next() {
			index := strings.Split(strings.Replace(string(iter.Key()), key+separator, "", 1), separator)
			if len(index) == 1 && index[0] != key {
				var newObject Object
				err := json.Unmarshal(iter.Value(), &newObject)
				if err == nil {
					res = append(res, newObject)
				} else {
					stderr.Println("getDataError:", err)
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

func (app *App) setData(mode string, key string, dataIndex string, subIndex string, data string) string {
	// TODO: retry
	now := time.Now().UTC().UnixNano()
	updated := now
	index := dataIndex
	if dataIndex == "" {
		index = strconv.FormatInt(now, 16) + subIndex
	}
	if mode == "mo" {
		key += separator + index
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
	stderr.Println("setDataError:", err)

	return ""
}

func (app *App) delData(mode string, key string, index string) {
	// TODO: retry
	if mode == "mo" {
		key += separator + index
	}

	err := app.db.Delete([]byte(key), nil)
	if err == nil {
		stdout.Println("deleted:", key)
		return
	}

	stderr.Println("delDataError:", err)
	return
}

func (app *App) getEncodedData(mode string, key string) string {
	raw := app.getData(mode, key)
	data := ""
	if len(raw) > 0 {
		data = base64.StdEncoding.EncodeToString(raw)
	}

	return data
}

func writeToClient(client *websocket.Conn, data string) {
	err := client.WriteMessage(1, []byte("{"+
		"\"data\": \""+data+"\""+
		"}"))
	if err != nil {
		stderr.Println("sendDataError:", err)
	}
}

func (app *App) sendData(clients []int) {
	if len(clients) > 0 {
		for _, clientIndex := range clients {
			data := app.getEncodedData(app.clients[clientIndex].mode, app.clients[clientIndex].key)
			for _, client := range app.clients[clientIndex].connections {
				writeToClient(client, data)
			}
		}
	}
}

func (app *App) sendTime(clients []*websocket.Conn) {
	now := time.Now().UTC().UnixNano()
	data := strconv.FormatInt(now, 10)
	for _, client := range clients {
		err := client.WriteMessage(1, []byte("{"+
			"\"data\": \""+data+"\""+
			"}"))
		if err != nil {
			stderr.Println("sendTimeError:", err)
		}
	}
}

func (app *App) findPool(mode string, key string) int {
	poolIndex := -1
	for i := range app.clients {
		if app.clients[i].key == key && app.clients[i].mode == mode {
			poolIndex = i
		}
	}

	return poolIndex
}

func removeLastIndex(key string) string {
	sp := strings.Split(key, separator)
	return strings.Replace(key, separator+sp[len(sp)-1], "", 1)
}

func (app *App) findConnections(mode string, key string) []int {
	var res []int
	for i := range app.clients {
		if (app.clients[i].key == key && app.clients[i].mode == mode) ||
			(mode == "sa" && app.clients[i].mode == "mo" && removeLastIndex(key) == app.clients[i].key) ||
			(mode == "mo" && app.clients[i].mode == "sa" && removeLastIndex(app.clients[i].key) == key) {
			res = append(res, i)
		}
	}

	return res
}

func (app *App) findClient(poolIndex int, client *websocket.Conn) int {
	clientIndex := -1
	for i := range app.clients[poolIndex].connections {
		if app.clients[poolIndex].connections[i] == client {
			clientIndex = i
		}
	}

	return clientIndex
}

func (app *App) newClient(w http.ResponseWriter, r *http.Request, mode string, key string) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		// define the upgrade success
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Upgrade") == "websocket"
		},
	}

	client, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		stderr.Println("socketUpgradeError:", err)
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

	stdout.Println("socket clients:", app.clients[poolIndex])
	return client, nil
}

func (app *App) closeClient(client *websocket.Conn, mode string, key string) {
	// remove the client before closing
	stdout.Println("socket client closing:", key)
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
		stdout.Println("socket pool empty:", key)
	}
	client.Close()
}

func (app *App) readClient(client *websocket.Conn, mode string, key string) {
	for {
		_, message, err := client.ReadMessage()

		if err != nil {
			stderr.Println("readSocketError:", err)
			break
		}

		var wsEvent map[string]interface{}
		err = json.Unmarshal(message, &wsEvent)
		if err != nil {
			stderr.Println("jsonUnmarshalError:", err)
			break
		}
		json.Unmarshal(message, &wsEvent)
		poolIndex := app.findPool(mode, key)
		clientIndex := app.findClient(poolIndex, client)
		op := ""
		index := ""
		data := ""
		if wsEvent["op"] != nil {
			op = wsEvent["op"].(string)
		}
		if wsEvent["index"] != nil {
			index = wsEvent["index"].(string)
		}
		if wsEvent["data"] != nil {
			data = wsEvent["data"].(string)
		}

		switch op {
		case "":
			if data != "" {
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

func (app *App) wss(mode string) func(w http.ResponseWriter, r *http.Request) {
	// https://stackoverflow.com/a/19395288/6582356
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		// https://stackoverflow.com/a/26792316/6582356
		if strings.Contains(vars["key"], separator+separator) || strings.HasSuffix(vars["key"], separator) {
			stderr.Println("socketKeyError:", vars["key"])
			return
		}
		stdout.Println("new websocket client")
		stdout.Println("mode:", mode)
		stdout.Println("key:", vars["key"])
		client, err := app.newClient(w, r, mode, vars["key"])

		if err != nil {
			stderr.Println("socketUpgradeError:", err)
			return
		}

		// send initial msg
		data := app.getEncodedData(mode, vars["key"])
		writeToClient(client, data)

		// defered client close
		defer app.closeClient(client, mode, vars["key"])

		app.readClient(client, mode, vars["key"])
	}
}

func (app *App) rPost(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars["key"]
		var obj Object
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		err := decoder.Decode(&obj)
		if err == nil {
			index := app.setData(mode, key, obj.Index, "R", obj.Data)
			app.sendData(app.findConnections(mode, key))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "{"+
				"\"index\": \""+index+"\""+
				"}")
			return
		}

		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "%s", err)
	}
}

func (app *App) rGet(mode string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars["key"]
		data := string(app.getData(mode, key))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, data)
	}
}

func (app *App) timeWs(w http.ResponseWriter, r *http.Request) {
	mode := "ws"
	key := "time"
	stdout.Println("new timer client")
	client, err := app.newClient(w, r, mode, key)

	if err != nil {
		stderr.Println("socketUpgradeError:", err)
		return
	}

	defer app.closeClient(client, mode, key)

	for {
		_, _, err := client.ReadMessage()

		if err != nil {
			stderr.Println("readSocketError:", err)
			break
		}
	}

	app.sendTime([]*websocket.Conn{client})
}

func (app *App) initialize(host *string, port *int) {
	app.Router = mux.NewRouter()
	app.address = *host + ":" + strconv.Itoa(*port)
	// https://stackoverflow.com/a/51681675/6582356
	stdout = log.New(
		os.Stdout,
		color.BBlue("SAMO~[")+
			color.BPurple(time.Now().String())+
			color.BBlue("]"),
		0)
	stderr = log.New(
		os.Stderr,
		color.BRed("SAMO~[")+
			color.BPurple(time.Now().String())+
			color.BRed("]"),
		0)
	app.Router.HandleFunc("/", app.getStats)
	app.Router.HandleFunc("/sa/{key:[a-zA-Z0-9-:/]+$}", app.wss("sa"))
	app.Router.HandleFunc("/mo/{key:[a-zA-Z0-9-:/]+$}", app.wss("mo"))
	app.Router.HandleFunc("/r/sa/{key:[a-zA-Z0-9-:/]+$}", app.rPost("sa")).Methods("POST")
	app.Router.HandleFunc("/r/mo/{key:[a-zA-Z0-9-:/]+$}", app.rPost("mo")).Methods("POST")
	app.Router.HandleFunc("/r/sa/{key:[a-zA-Z0-9-:/]+$}", app.rGet("sa")).Methods("GET")
	app.Router.HandleFunc("/r/mo/{key:[a-zA-Z0-9-:/]+$}", app.rGet("mo")).Methods("GET")
	app.Router.HandleFunc("/time", app.timeWs)
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	app := App{}
	app.initialize(host, port)
	var err error
	app.db, err = leveldb.OpenFile(*storage, nil)
	defer app.db.Close()
	if err == nil {
		stdout.Println("Listening on", app.address)
		ticker := time.NewTicker(time.Second)
		quit := make(chan struct{})
		go func() {
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
		}()
		handler := cors.Default().Handler(app.Router)
		log.Fatal(http.ListenAndServe(app.address, handler))
	}
}
