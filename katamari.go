package katamari

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/benitogf/coat"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/benitogf/katamari/stream"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const deadlineMsg = "katamari: server deadline reached"

// audit requests function
// will define approval or denial by the return value
// r: the request to be audited
// returns
// true: approve the request
// false: deny the request
type audit func(r *http.Request) bool

// Server application
//
// Router: can be predefined with routes and passed to be extended
//
// NoBroadcastKeys: array of keys that should not broadcast on changes
//
// DbOpt: options for storage
//
// Audit: function to audit requests
//
// Workers: number of workers to use as readers of the storage->broadcast channel
//
// ForcePatch: flag to force patch operations
//
// OnSubscribe: function to monitor subscribe events
//
// OnUnsubscribe: function to monitor unsubscribe events
//
// OnClose: function that triggers before closing the application
//
// Deadline: time duration of a request before timing out
//
// AllowedOrigins: list of allowed origins for cross domain access, defaults to ["*"]
//
// Storage: database interdace implementation
//
// Silence: output silence flag
//
// Static: static routing flag
//
// Tick: time interval between ticks on the clock subscription
//
// NamedSocket: name of the ipc socket
//
// Signal: os signal channel
//
// Client: http client to make requests
type Server struct {
	wg              sync.WaitGroup
	server          *http.Server
	Router          *mux.Router
	Stream          stream.Pools
	filters         filters
	Pivot           string
	NoBroadcastKeys []string
	DbOpt           interface{}
	Audit           audit
	Workers         int
	ForcePatch      bool
	OnSubscribe     stream.Subscribe
	OnUnsubscribe   stream.Unsubscribe
	OnClose         func()
	Deadline        time.Duration
	AllowedOrigins  []string
	Storage         Database
	Address         string
	closing         int64
	active          int64
	Silence         bool
	Static          bool
	Tick            time.Duration
	console         *coat.Console
	Signal          chan os.Signal
	Client          *http.Client
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (app *Server) waitListen() {
	var err error
	err = app.Storage.Start(StorageOpt{
		NoBroadcastKeys: app.NoBroadcastKeys,
		DbOpt:           app.DbOpt,
	})
	if err != nil {
		log.Fatal(err)
	}
	app.server = &http.Server{
		WriteTimeout:      1 * time.Minute,
		ReadTimeout:       1 * time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       10 * time.Second,
		Addr:              app.Address,
		Handler: cors.New(cors.Options{
			AllowedMethods: []string{"GET", "POST", "DELETE", "PUT"},
			AllowedOrigins: app.AllowedOrigins,
			AllowedHeaders: []string{"Authorization", "Content-Type"},
			// AllowCredentials: true,
			// Debug:          true,
		}).Handler(handlers.CompressHandler(app.Router))}
	ln, err := net.Listen("tcp4", app.Address)
	if err != nil {
		log.Fatal("failed to start tcp, ", err)
	}
	app.Address = ln.Addr().String()
	atomic.StoreInt64(&app.active, 1)
	app.wg.Done()
	err = app.server.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
	if atomic.LoadInt64(&app.closing) != 1 {
		log.Fatal(err)
	}
}

// Active check if the server is active
func (app *Server) Active() bool {
	return atomic.LoadInt64(&app.active) == 1 && atomic.LoadInt64(&app.closing) == 0
}

func (app *Server) waitStart() {
	if atomic.LoadInt64(&app.active) == 0 || !app.Storage.Active() {
		log.Fatal("server start failed")
	}

	for i := 0; i < app.Workers; i++ {
		go app.watch(app.Storage.Watch())
	}

	app.console.Log("glad to serve[" + app.Address + "]")
}

// ForceFetch data, update cache and apply filter
func (app *Server) ForceFetch(key string, filter string) (stream.Cache, error) {
	var err error
	raw, _ := app.Storage.Get(key)
	if len(raw) == 0 {
		raw = objects.EmptyObject
	}
	// newVersion := app.Stream.SetCache(key, raw)
	newVersion := time.Now().UTC().Unix()
	cache := stream.Cache{
		Version: newVersion,
		Data:    raw,
	}
	cache.Data, err = app.filters.Read.check(filter, cache.Data, app.Static)
	if err != nil {
		return cache, err
	}
	return cache, nil
}

// Fetch data, update cache and apply filter
func (app *Server) Fetch(key string, filter string) (stream.Cache, error) {
	cache, err := app.Stream.GetCache(key)
	if err != nil {
		raw, _ := app.Storage.Get(key)
		if len(raw) == 0 {
			raw = objects.EmptyObject
		}
		newVersion := app.Stream.SetCache(key, raw)
		cache = stream.Cache{
			Version: newVersion,
			Data:    raw,
		}
	}
	cache.Data, err = app.filters.Read.check(filter, cache.Data, app.Static)
	if err != nil {
		return cache, err
	}
	return cache, nil
}

// getPatch of the by a poolIndex, it returns
//
// - message (string)
//
// - bool flag to indicate if the message is a full snapshot
//
// - version timestamp
//
// - error
func (app *Server) getPatch(poolIndex int) (string, bool, int64, error) {
	raw, _ := app.Storage.Get(app.Stream.Pools[poolIndex].Key)
	if len(raw) == 0 {
		raw = objects.EmptyObject
	}
	filteredData, err := app.filters.Read.check(
		app.Stream.Pools[poolIndex].Filter, raw, app.Static)
	if err != nil {
		return "", false, 0, err
	}
	modifiedData, snapshot, version := app.Stream.Patch(poolIndex, filteredData)
	return messages.Encode(modifiedData), snapshot, version, nil
}

func (app *Server) broadcast(key string) {
	app.Stream.UseConnections(key, func(poolIndex int) {
		data, snapshot, version, err := app.getPatch(poolIndex)
		if err != nil {
			return
		}
		app.Stream.Broadcast(poolIndex, data, snapshot, version)
	})
}

func (app *Server) watch(sc StorageChan) {
	for {
		ev := <-sc
		if ev.Key != "" {
			app.console.Log("broadcast[" + ev.Key + "]")
			go app.broadcast(ev.Key)
		}
		if !app.Storage.Active() {
			break
		}
	}
}

// defaults will populate the server fields with their zero values
func (app *Server) defaults() {
	if app.Router == nil {
		app.Router = mux.NewRouter()
	}

	if app.Deadline.Nanoseconds() == 0 {
		app.Deadline = time.Second * 10
	}

	if app.OnClose == nil {
		app.OnClose = func() {}
	}

	if app.AllowedOrigins == nil || len(app.AllowedOrigins) == 0 {
		app.AllowedOrigins = []string{"*"}
	}

	if app.console == nil {
		app.console = coat.NewConsole(app.Address, app.Silence)
	}

	if app.Stream.Console == nil {
		app.Stream.Console = app.console
	}

	if app.Storage == nil {
		app.Storage = &MemoryStorage{}
	}

	if app.Tick == 0 {
		app.Tick = 1 * time.Second
	}

	if app.Audit == nil {
		app.Audit = func(r *http.Request) bool { return true }
	}

	if app.OnSubscribe == nil {
		app.OnSubscribe = func(key string) error { return nil }
	}

	if app.Stream.OnSubscribe == nil {
		app.Stream.OnSubscribe = app.OnSubscribe
	}

	if app.OnUnsubscribe == nil {
		app.OnUnsubscribe = func(key string) {}
	}

	if app.Stream.OnUnsubscribe == nil {
		app.Stream.OnUnsubscribe = app.OnUnsubscribe
	}

	if app.Workers == 0 {
		app.Workers = 6
	}

	if app.NoBroadcastKeys == nil {
		app.NoBroadcastKeys = []string{}
	}

	if app.Client == nil {
		app.Client = &http.Client{
			Timeout: 800 * time.Millisecond,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   800 * time.Millisecond,
					KeepAlive: 5 * time.Second,
				}).Dial,
				IdleConnTimeout:       10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				MaxConnsPerHost:       3000,
				MaxIdleConns:          10000,
				MaxIdleConnsPerHost:   1000,
				DisableKeepAlives:     false,
			},
		}
	}

	app.Stream.ForcePatch = app.ForcePatch
	if len(app.Stream.Pools) == 0 {
		app.Stream.Pools = append(
			app.Stream.Pools,
			&stream.Pool{Key: ""})
	}
}

// Start : initialize and start the http server and database connection
func (app *Server) Start(address string) {
	app.Address = address
	if atomic.LoadInt64(&app.active) == 1 {
		app.console.Err("server already active")
		return
	}
	atomic.StoreInt64(&app.active, 0)
	atomic.StoreInt64(&app.closing, 0)
	app.defaults()
	// https://ieftimov.com/post/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation/
	app.Router.HandleFunc("/", app.getStats).Methods("GET")
	// https://www.calhoun.io/why-cant-i-pass-this-function-as-an-http-handler/
	app.Router.Handle("/{key:[a-zA-Z\\*\\d\\/]+}", http.TimeoutHandler(
		http.HandlerFunc(app.unpublish), app.Deadline, deadlineMsg)).Methods("DELETE")
	app.Router.Handle("/{key:[a-zA-Z\\*\\d\\/]+}", http.TimeoutHandler(
		http.HandlerFunc(app.publish), app.Deadline, deadlineMsg)).Methods("POST")
	app.Router.HandleFunc("/{key:[a-zA-Z\\*\\d\\/]+}", app.read).Methods("GET")
	app.Router.HandleFunc("/{key:[a-zA-Z\\*\\d\\/]+}", app.read).Queries("v", "{[\\d]}").Methods("GET")
	app.wg.Add(1)
	go app.waitListen()
	app.wg.Wait()
	app.waitStart()
	app.console = coat.NewConsole(app.Address, app.Silence)
	go app.tick()
}

// Close : shutdown the http server and database connection
func (app *Server) Close(sig os.Signal) {
	if atomic.LoadInt64(&app.closing) != 1 {
		atomic.StoreInt64(&app.closing, 1)
		atomic.StoreInt64(&app.active, 0)
		app.Storage.Close()
		app.OnClose()
		app.console.Err("shutdown", sig)
		if app.server != nil {
			app.server.Shutdown(context.Background())
		}
	}
}

// WaitClose : Blocks waiting for SIGINT, SIGTERM, SIGKILL, SIGHUP
func (app *Server) WaitClose() {
	app.Signal = make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(app.Signal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-app.Signal
		app.Close(sig)
		done <- true
	}()
	<-done
}
