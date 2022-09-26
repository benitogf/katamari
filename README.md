# katamari

[![Test](https://github.com/benitogf/katamari/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/benitogf/katamari/actions/workflows/tests.yml)

![katamari](katamari.jpg)

Zero configuration data persistence and communication layer.

Web service that behaves like a distributed filesystem in the sense that all routes are open by default, oposite to rails like frameworks where the user must define the routes before being able to interact with them.

Provides a dynamic websocket and restful http service to quickly prototype realtime applications, the interface has no fixed data structure or access regulations by default, to restrict access see: [define limitations](https://github.com/benitogf/katamari#control).

## features

- dynamic routing
- glob pattern routes
- [patch](http://jsonpatch.com) updates on subscriptions
- version check on subscriptions (no message on version match)
- restful CRUD service that reflects interactions to real-time subscriptions
- storage interfaces for memory only or leveldb and memory
- filtering and audit middleware
- auto managed timestamps (created, updated)

## quickstart

### client

There's a [js client](https://www.npmjs.com/package/katamari-client).

### server

with [go installed](https://golang.org/doc/install) get the library

```bash
go get github.com/benitogf/katamari
```

create a file `main.go`
```golang
package main

import "github.com/benitogf/katamari"

func main() {
  app := katamari.Server{}
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


# control

### static routes

Activating this flag will limit the server to process requests defined in read and write filters

```golang
app := katamari.Server{}
app.Static = true
```


### filters

- Write filters will be called before processing a write operation
- Read filters will be called before sending the results of a read operation
- if the static flag is enabled only filtered routes will be available

```golang
app.WriteFilter("books/*", func(index string, data json.RawMessage) (json.RawMessage, error) {
  // returning an error will deny the write
  return data, nil
})
app.ReadFilter("books/taup", func(index string, data json.RawMessage) (json.RawMessage, error) {
  // returning an error will deny the read
  return json.RawMessage(`{"intercepted":true}`), nil
})
app.DeleteFilter("books/taup", func(key string) (error) {
  // returning an error will prevent the delete
  return errors.New("can't delete")
})
```

### audit

```golang
app.Audit = func(r *http.Request) bool {
  return false // condition to allow access to the resource
}
```

### subscribe events capture

```golang
// new subscription event
server.OnSubscribe = func(key string) error {
  log.Println(key)
  // returning an error will deny the subscription
  return nil
}
// closing subscription event
server.OnUnsubscribe = func(key string) {
  log.Println(key)
}
```

### extra routes

```golang
// Define custom endpoints
app.Router = mux.NewRouter()
app.Router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  fmt.Fprintf(w, "{}")
})
app.Start("localhost:8800")
```

# libraries

- [jwt authentication](https://github.com/benitogf/auth)
- [leveldb storage](https://github.com/benitogf/level)
- [pebble storage](https://github.com/benitogf/pebble)
- [inmemory with leveldb persistence storage](https://github.com/benitogf/lvlmap)
- [distiribution adapter](https://github.com/benitogf/pivot)