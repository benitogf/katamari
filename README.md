# samo

[![Build Status][build-image]][build-url]


[build-url]: https://travis-ci.com/benitogf/samo
[build-image]: https://api.travis-ci.com/benitogf/samo.svg?branch=master&style=flat-square

Zero configuration data persistence and communication layer.

Web service that behaves like a distributed filesystem in the sense that all routes are open by default, oposite to rails like frameworks where the user must define the routes before being able to interact with them.

Provides a dynamic websocket and restful http service to quickly prototype realtime applications, the interface has no fixed data structure or access regulations by default, to restrict access see: [define limitations](https://github.com/benitogf/samo#filters-audit-subscription-events-and-extra-routes).

## features

- dynamic routing
- glob pattern routes
- [patch](http://jsonpatch.com) updates on subscriptions
- restful CRUD service that reflects interactions to real-time subscriptions
- storage interfaces for memory, leveldb, and etcd
- filtering and audit middleware
- auto managed timestamps (created, updated)

# quickstart

## client

There's a [js client](https://www.npmjs.com/package/samo-js-client).

## server

download a [release](https://github.com/benitogf/samo/releases) or with [go installed](https://golang.org/doc/install) get the library:

```bash
go get github.com/benitogf/samo
```

then create a file `main.go` as:
```golang
package main

import "github.com/benitogf/samo"

func main() {
	app := samo.Server{}
	app.Start("localhost:8800")
	app.WaitClose()
}
```

finally run the service:
```bash
go run main.go
```

# routes

| method | description | url    |
| ------------- |:-------------:| -----:|
| GET | key list | http://{host}:{port} |
| websocket| time ticker | ws://{host}:{port} |
| POST | create/update | http://{host}:{port}/{key} |
| GET | read | http://{host}:{port}/{key} |
| DELETE | delete | http://{host}:{port}/{key} |
| websocket| subscribe | ws://{host}:{port}/{key} |

# filters, audit, subscription events and extra routes

    Define ad lib receive and send filter criteria using key glob patterns, audit middleware, and extra routes

### filters

```go
	// Filters
	app.ReceiveFilter("bag/*", func(index string, data []byte) ([]byte, error) {
		if string(data) != "marbles" {
			return nil, errors.New("filtered")
		}

		return data, nil
	})
	app.SendFilter("bag/1", func(index string, data []byte) ([]byte, error) {
		return []byte("intercepted"), nil
	})
```

### audit

```go
	// Audit requests
	app.Audit = func(r *http.Request) bool {
		return r.Method == "GET" && r.Header.Get("Upgrade") != "websocket"
  }
```

### subscribe

```go
	// Subscription events
	server.Subscribe = func(mode string, key string, remoteAddr string) error {
		log.Println(mode, key)
		return nil
	}
	server.Unsubscribe = func(mode string, key string, remoteAddr string) {
		log.Println(mode, key)
	}
```

### extra routes

```go
	// Predefine the router
	app.Router = mux.NewRouter()
	app.Router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "123")
	})
```

# data persistence layer

    Use alternative storages (the default is memory)

### level
```go
package main

import "github.com/benitogf/samo"

func main() {
	app := samo.Server{}
	app.Storage = &samo.LevelStorage{
		Path:    "data/db"}
	app.Start("localhost:8800")
	app.WaitClose()
}
```
### etcd
```go
package main

import "github.com/benitogf/samo"

func main() {
	app := samo.Server{}
	app.Storage = &samo.EtcdStorage{
            Path:    "data/default.etcd",
            Peers: []string{"localhost:2379"}}
	app.Start("localhost:8800")
	app.WaitClose()
}
```


