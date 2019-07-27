# samo

[![Build Status][build-image]][build-url]


[build-url]: https://travis-ci.com/benitogf/samo
[build-image]: https://api.travis-ci.com/benitogf/samo.svg?branch=master&style=flat-square

Zero configuration data persistence and communication layer for your application.

Provides a dynamic websocket and restful http service to quickly prototype realtime applications, the interface has no fixed data structure or access regulations by default, to restrict access see: [define limitations](https://github.com/benitogf/samo#filters-audit-subscription-events-and-extra-routes).

As stated in this relevant [article](https://medium.com/@brenda.clark/firebase-alternative-3-open-source-ways-to-follow-e45d9347bc8c) there are some similar solutions.

## features

- dynamic routing
- glob pattern subscriptions
- restful CRUD service that reflects interactions to real-time subscriptions
- storage interfaces for leveldb, redis, mongodb, mariadb, and etcd
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

# Specs

The service expose four kinds of routes:

- /time: this route will send a timestamp at a configurable interval, default is 1 second

- /r/**: restful CRUD api

- /sa/**: single allocation allows subscriptions to a key->value.

- /mo/**: multiple objects allows subscription to a prefix key/*->list. glob patterns allowed.

## general routes

| method | description | url    |
| ------------- |:-------------:| -----:|
| GET | key list | http://{host}:{port} |
| DELETE | del | http://{host}:{port}/r/{key} |
| websocket| time ticker | ws://{host}:{port}/time |

## single allocation (sa)

will handle the key as key->value

| method | description | url    |
| ------------- |:-------------:| -----:|
| websocket| key data events: update, delete | ws://{host}:{port}/sa/{key} |
| POST | create/update | http://{host}:{port}/r/sa/{key} |
| GET | get object | http://{host}:{port}/r/sa/{key} |

## multiple objects (mo)

will handle the key as a prefix to get a list of every key/[index...], excluding the empty index (key->value)

| method  | description | url    |
| ------------- |:-------------:| -----:|
| websocket | key data events: new, update, delete, glob patterns allowed | ws://{host}:{port}/mo/{key} |
| POST | create/update, if the index is not provided it will autogenerate a new one, preexistent data on the provided key/index will be overwriten | http://{host}:{port}/r/mo |
| GET | get list | http://{host}:{port}/r/mo/{key} |

## filters, audit, subscription events and extra routes

    Define ad lib receive and send filter criteria using key glob patterns, audit middleware, subscription events, and extra routes

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/benitogf/samo"
	"github.com/gorilla/mux"
)

func main() {
	app := samo.Server{}
	app.Static = true // limit to filtered paths

	// Audit requests
	app.Audit = func(r *http.Request) bool {
		return r.Method == "GET" && r.Header.Get("Upgrade") != "websocket"
  }

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

	// Subscription events
	server.Subscribe = func(mode string, key string, remoteAddr string) error {
		log.Println(mode, key)
		return nil
	}
	server.Unsubscribe = func(mode string, key string, remoteAddr string) {
		log.Println(mode, key)
	}
	// Predefine the router
	app.Router = mux.NewRouter()
	app.Router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "123")
	})
	app.Start("localhost:8800")
	app.WaitClose()
}
```

## data persistence layer

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
### redis
```go
package main

import "github.com/benitogf/samo"

func main() {
	app := samo.Server{}
	app.Storage = &samo.RedisStorage{
		Address: "localhost:6379",
		Password: ""}
	app.Start("localhost:8800")
	app.WaitClose()
}
```
### mongo
```go
package main

import "github.com/benitogf/samo"

func main() {
	app := samo.Server{}
	app.Storage = &samo.MongoStorage{
		Address: "localhost:27017"}
	app.Start("localhost:8800")
	app.WaitClose()
}
```
### etcd
with this storage an embeded server will run in tandem
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


