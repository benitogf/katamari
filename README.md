# samo

[![Build Status][build-image]][build-url]


[build-url]: https://travis-ci.com/benitogf/samo
[build-image]: https://api.travis-ci.com/benitogf/samo.svg?branch=master&style=flat-square

Zero configuration data persistence and communication layer.

Web service that behaves like a distributed filesystem in the sense that all routes are open by default, oposite to rails like frameworks where the user must define the routes before being able to interact with them.

Provides a dynamic websocket and restful http service to quickly prototype realtime applications, the interface has no fixed data structure or access regulations by default, to restrict access see: [define limitations](https://github.com/benitogf/samo#creating-rules-and-control).

## features

- dynamic routing
- glob pattern routes
- [patch](http://jsonpatch.com) updates on subscriptions
- restful CRUD service that reflects interactions to real-time subscriptions
- named socket ipc
- storage interfaces for memory, leveldb, and etcd
- filtering and audit middleware
- auto managed timestamps (created, updated)

## quickstart

### client

There's a [js client](https://www.npmjs.com/package/samo-js-client).

### server

with [go installed](https://golang.org/doc/install) get the library

```bash
go get github.com/benitogf/samo
```

create a file `main.go`
```golang
package main

import "github.com/benitogf/samo"

func main() {
  app := samo.Server{}
  app.Start("localhost:8800")
  app.WaitClose()
}
```

run the service:
```bash
go run main.go
```

# routes

| method | description | url    |
| ------------- |:-------------:| -----:|
| GET | key list | http://{host}:{port} |
| websocket| clock | ws://{host}:{port} |
| POST | create/update | http://{host}:{port}/{key} |
| GET | read | http://{host}:{port}/{key} |
| DELETE | delete | http://{host}:{port}/{key} |
| websocket| subscribe | ws://{host}:{port}/{key} |

# creating rules and control

    Define ad lib receive and send filter criteria using key glob patterns, audit middleware, and extra routes

Using the default open setting is usefull while prototyping, but maybe not ideal to deploy as a public service.

A one route server example:

```golang
package main

import "github.com/benitogf/samo"
import "net/http"


// if the filter is not defined for a route while static is enabled
// the route will return 400
func openFilter(index string, data []byte) ([]byte, error) {
  return data, nil
}

// perform audits on the request path/headers/referer
// if the function returns false the request will return
// status 401
func audit(r *http.Request) bool {
  // allow clock subscription
  if r.URL.Path == "/" {
    return true
  }
  if r.URL.Path == "/open" {
    return true
  }

  return false
}

func main() {
  app := samo.Server{}
  app.Static = true
  app.Audit = audit
  app.WriteFilter("open", openFilter)
  app.ReadFilter("open", openFilter)
  app.Start("localhost:8800")
  app.WaitClose()
}
```

### static routes

Activating this flag will limit the server to process requests defined in read and write filters

```golang
app := samo.Server{}
app.Static = true
```


### filters

- Write filters will be called before processing a write operation
- Read filters will be called before sending the results of a read operation
- if the static flag is enabled only filtered routes will be available

```golang
app.WriteFilter("books/*", func(index string, data []byte) ([]byte, error) {
  // returning an error will deny the write
  return data, nil
})
app.ReadFilter("books/taup", func(index string, data []byte) ([]byte, error) {
  // returning an error will deny the read
  return []byte("intercepted"), nil
})
```

### audit

```golang
app.Audit = func(r *http.Request) bool {
  return false // condition to allow access to the resource
}
```

### subscribe

```golang
// new subscription
server.Subscribe = func(key string) error {
  log.Println(key)
  // returning an error will deny the subscription
  return nil
}
// closing subscription
server.Unsubscribe = func(key string) {
  log.Println(key)
}
```

### extra routes

```golang
// Predefine the router
app.Router = mux.NewRouter()
app.Router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  fmt.Fprintf(w, "123")
})
app.Start("localhost:8800")
```

# ipc named socket

Subscribe on a separated process without websocket

### client

```go
package main

import (
	"log"

	"github.com/benitogf/nsocket"
)

func main() {
	client, err := nsocket.Dial("testns", "books/*")
	if err != nil {
		log.Fatal(err)
	}
	for {
		msg, err = client.Read()
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(msg)
	}
}
```

### server

```go
package main

import "github.com/benitogf/samo"

func main() {
  app := samo.Server{}
  app.NamedSocket = "testns" // set this field to the name to use
  app.Start("localhost:8800")
  app.WaitClose()
}
```

# data persistence layer

    Use alternative storages (the default is memory)

### level
```go
package main

import "github.com/benitogf/samo"

func main() {
  app := samo.Server{}
  app.Storage = &samo.LevelStorage{Path:"data/db"}
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


