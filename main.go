package samo

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/benitogf/coat"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// conn extends the websocket connection with a mutex
// https://godoc.org/github.com/gorilla/websocket#hdr-Concurrency
type conn struct {
	conn  *websocket.Conn
	mutex sync.Mutex
}

// pool of mode/key filtered websocket connections
type pool struct {
	key         string
	mode        string
	connections []*conn
}

// Audit : function to provide approval or denial of requests
type Audit func(r *http.Request) bool

// Server : SAMO application server
type Server struct {
	mutex        sync.RWMutex
	mutexClients sync.RWMutex
	server       *http.Server
	Router       *mux.Router
	clients      []*pool
	Filters      Filters
	Audit        Audit
	Storage      Database
	separator    string
	address      string
	closing      bool
	Silence      bool
	Static       bool
	console      *coat.Console
	objects      *Objects
	keys         *Keys
	messages     *Messages
}

// Stats : data structure of global keys
type Stats struct {
	Keys []string `json:"keys"`
}

func (app *Server) makeRouteRegex() string {
	return "[a-zA-Z\\d][a-zA-Z\\d\\" + app.separator + "]+[a-zA-Z\\d]"
}

func (app *Server) waitListen() {
	var err error
	err = app.Storage.Start(app.separator)
	if err == nil {
		app.mutex.Lock()
		app.server = &http.Server{
			Addr: app.address,
			Handler: cors.New(cors.Options{
				AllowedMethods: []string{"GET", "POST", "DELETE"},
				// AllowedOrigins: []string{"http://foo.com", "http://foo.com:8080"},
				// AllowCredentials: true,
				// Debug: true,
			}).Handler(app.Router)}
		app.mutex.Unlock()
		err = app.server.ListenAndServe()
		if !app.closing {
			log.Fatal(err)
		}
		return
	}

	log.Fatal(err)
}

func (app *Server) waitStart() {
	tryes := 0
	app.mutex.RLock()
	for (app.server == nil || !app.Storage.Active()) && tryes < 1000 {
		tryes++
		app.mutex.RUnlock()
		time.Sleep(10 * time.Millisecond)
		app.mutex.RLock()
	}
	app.mutex.RUnlock()
	if app.server == nil || !app.Storage.Active() {
		log.Fatal("Server start failed")
	}
	app.console.Log("glad to serve[" + app.address + "]")
}

// Start : initialize and start the http server and database connection
// 	port : service port 8800
//  host : service host "localhost"
// 	storage : path to the storage folder "data/db"
// 	separator : rune to use as key separator '/'
func (app *Server) Start(address string) {
	app.closing = false
	// app.objects = &Objects{&Keys{}}
	app.address = address
	if app.separator == "" || len(app.separator) > 1 {
		app.separator = "/"
	}
	if app.Router == nil {
		app.Router = mux.NewRouter()
	}
	app.console = coat.NewConsole(app.address, app.Silence)
	if app.Storage == nil {
		app.Storage = &MemoryStorage{
			Memdb:   make(map[string][]byte),
			Storage: &Storage{Active: false}}
	}
	if app.Audit == nil {
		app.Audit = func(r *http.Request) bool { return true }
	}
	rr := app.makeRouteRegex()
	app.Router.HandleFunc("/", app.getStats)
	app.Router.HandleFunc("/r/{key:"+rr+"}", app.rDel).Methods("DELETE")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rPost("mo")).Methods("POST")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rGet("mo")).Methods("GET")
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rPost("sa")).Methods("POST")
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rGet("sa")).Methods("GET")
	app.Router.HandleFunc("/sa/{key:"+rr+"}", app.wss("sa"))
	app.Router.HandleFunc("/mo/{key:"+rr+"}", app.wss("mo"))
	app.Router.HandleFunc("/time", app.timeWs)
	go app.waitListen()
	app.waitStart()
	go app.timer()
}

// Close : shutdown the http server and database connection
func (app *Server) Close(sig os.Signal) {
	if !app.closing {
		app.closing = true
		app.Storage.Close()
		app.console.Err("shutdown", sig)
		if app.server != nil {
			app.server.Shutdown(context.Background())
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
