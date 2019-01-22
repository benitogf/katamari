package samo

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bclicn/color"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// Pool : mode/key based websocket connections and watcher
type Pool struct {
	key         string
	mode        string
	connections []*websocket.Conn
}

// Audit : function to provide approval or denial of requests
type Audit func(r *http.Request) bool

// Archetype : function to check proper key->data covalent bond
type Archetype func(data string) bool

// Archetypes : a map that allows structure and content formalization of key->data
type Archetypes map[string]Archetype

// Server : SAMO application server
type Server struct {
	Server     *http.Server
	Router     *mux.Router
	clients    []*Pool
	Archetypes Archetypes
	Audit      Audit
	storage    *Storage
	separator  string
	address    string
	console    *Console
	helpers    *Helpers
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

func (app *Server) waitStart() {
	tryes := 0
	for (app.Server == nil || !app.storage.active) && tryes < 100 {
		tryes++
		time.Sleep(100 * time.Millisecond)
	}
	if app.Server == nil || !app.storage.active {
		log.Fatal("Server start failed")
	}
}

// Start : initialize and start the http server and database connection
// 	port : service port 8800
//  host : service host "localhost"
// 	storage : path to the storage folder "data/db"
// 	separator : rune to use as key separator '/'
func (app *Server) Start(address string, storage string, separator rune) {
	app.address = address
	// TODO: other kinds of storage
	// redis, memory, etc
	app.storage = &Storage{
		path:   storage,
		kind:   "leveldb",
		active: false}
	app.separator = string(separator)
	app.Router = mux.NewRouter()
	app.helpers = &Helpers{}
	app.closing = false
	if app.Audit == nil {
		app.Audit = func(r *http.Request) bool { return true }
	}
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
	rr := app.helpers.makeRouteRegex(app.separator)
	app.Router.HandleFunc("/", app.getStats)
	app.Router.HandleFunc("/r/{key:"+rr+"}", app.rDel).Methods("DEL")
	app.Router.HandleFunc("/sa/{key:"+rr+"}", app.wss("sa"))
	app.Router.HandleFunc("/mo/{key:"+rr+"}", app.wss("mo"))
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rPost("sa")).Methods("POST")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rPost("mo")).Methods("POST")
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rGet("sa")).Methods("GET")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rGet("mo")).Methods("GET")
	app.Router.HandleFunc("/time", app.timeWs)
	go func() {
		var err error
		app.console.log("starting db")
		err = app.storage.start(app.console, app.separator)
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
	app.waitStart()
	app.console.log("glad to serve[" + app.address + "]")
	go app.timer()
}

// Close : shutdowns the http server and database connection
func (app *Server) Close(sig os.Signal) {
	if !app.closing {
		app.closing = true
		app.storage.close()
		app.console.err("shutdown", sig)
		if app.Server != nil {
			app.Server.Shutdown(context.Background())
		}
	}
}

// WaitClose : Blocks waiting for SIGINT or SIGTERM
func (app *Server) WaitClose() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		app.Close(sig)
		done <- true
	}()
	<-done
}
