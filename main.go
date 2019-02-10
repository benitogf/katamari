package samo

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/benitogf/coat"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Audit : function to provide approval or denial of requests
type Audit func(r *http.Request) bool

// Server : SAMO application server
type Server struct {
	mutex       sync.RWMutex
	server      *http.Server
	Router      *mux.Router
	stream      stream
	Filters     Filters
	Audit       Audit
	Subscribe   Subscribe
	Unsubscribe Unsubscribe
	Storage     Database
	separator   string
	address     string
	closing     int64
	active      int64
	Silence     bool
	Static      bool
	console     *coat.Console
	objects     *Objects
	keys        *Keys
	messages    *Messages
}

func (app *Server) makeRouteRegex() string {
	return "[a-zA-Z\\d][a-zA-Z\\d\\" + app.separator + "]+[a-zA-Z\\d]"
}

func (app *Server) waitListen() {
	var err error
	err = app.Storage.Start()
	if err != nil {
		log.Fatal(err)
	}

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
	if atomic.LoadInt64(&app.closing) != 1 {
		log.Fatal(err)
	}
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
		log.Fatal("server start failed")
	}
	app.console.Log("glad to serve[" + app.address + "]")
}

// Start : initialize and start the http server and database connection
func (app *Server) Start(address string) {
	if atomic.LoadInt64(&app.active) == 1 {
		app.console.Err("server already active")
		return
	}

	atomic.StoreInt64(&app.active, 1)
	atomic.StoreInt64(&app.closing, 0)
	app.address = address
	if app.separator == "" || len(app.separator) > 1 {
		app.separator = "/"
	}

	if app.Router == nil {
		app.Router = mux.NewRouter()
	}
	app.console = coat.NewConsole(app.address, app.Silence)
	app.stream.console = app.console

	if app.Storage == nil {
		app.Storage = &MemoryStorage{
			Storage: &Storage{Separator: app.separator}}
	}
	if app.Audit == nil {
		app.Audit = func(r *http.Request) bool { return true }
	}
	if app.Subscribe == nil {
		app.Subscribe = func(mode string, key string, remoteAddr string) error { return nil }
	}
	if app.Unsubscribe == nil {
		app.Unsubscribe = func(mode string, key string, remoteAddr string) {}
	}
	app.stream.Subscribe = app.Subscribe
	app.stream.Unsubscribe = app.Unsubscribe
	rr := app.makeRouteRegex()
	app.Router.HandleFunc("/", app.getStats)
	app.Router.HandleFunc("/r/{key:"+rr+"}", app.rDel).Methods("DELETE")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rPost("mo")).Methods("POST")
	app.Router.HandleFunc("/r/mo/{key:"+rr+"}", app.rGet("mo")).Methods("GET")
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rPost("sa")).Methods("POST")
	app.Router.HandleFunc("/r/sa/{key:"+rr+"}", app.rGet("sa")).Methods("GET")
	app.Router.HandleFunc("/sa/{key:"+rr+"}", app.ws("sa"))
	app.Router.HandleFunc("/mo/{key:"+rr+"}", app.ws("mo"))
	app.Router.HandleFunc("/time", app.clock)
	go app.waitListen()
	app.waitStart()
	go app.tick()
}

// Close : shutdown the http server and database connection
func (app *Server) Close(sig os.Signal) {
	if atomic.LoadInt64(&app.closing) != 1 {
		atomic.StoreInt64(&app.closing, 1)
		atomic.StoreInt64(&app.active, 0)
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
