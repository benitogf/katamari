package client

import (
	"net/url"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/gorilla/websocket"
)

// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#No-parameterized-methods
type Message interface {
	objects.Object | []objects.Object
}

// type onMessageCallback [T Message]func(*katamari.Server, T)
// type onMessageCallback func(*katamari.Server, interface{})

type ListCallback func(*katamari.Server, []objects.Object)
type Callback func(*katamari.Server, objects.Object)

type Reader struct {
	ListCallback ListCallback
	Callback     Callback
}

func Subscribe(server *katamari.Server, protocol, host, path string, reader Reader) {
	retryCount := 0
	cache := ""
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"
	for {
		wsURL := url.URL{Scheme: protocol, Host: host, Path: path}
		wsClient, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
		if wsClient == nil || err != nil {
			server.Console.Err("failed websocket dial ", err)
			time.Sleep(2 * time.Second)
			continue
		}
		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil || message == nil {
				server.Console.Err("failed websocket connection ", err)
				wsClient.Close()
				break
			}

			if isList {
				var objs []objects.Object
				cache, objs, err = messages.PatchList(message, cache)
				if err != nil {
					server.Console.Err("fail to parse message from websocket", err)
					break
				}
				retryCount = 0
				reader.OnListUpdate(server, objs)
				continue
			}

			var obj objects.Object
			cache, obj, err = messages.Patch(message, cache)
			if err != nil {
				server.Console.Err("fail to parse message from websocket", err)
				break
			}

			retryCount = 0
			reader.OnUpdate(server, obj)
		}
		retryCount++
		if retryCount < 30 {
			server.Console.Err("reconnecting websocket...", err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		if retryCount < 100 {
			server.Console.Err("reconnecting websocket in 2 seconds...", err)
			time.Sleep(2 * time.Second)
			continue
		}

		server.Console.Err("reconnecting websocket in 10 seconds...", err)
		time.Sleep(10 * time.Second)
	}
}
