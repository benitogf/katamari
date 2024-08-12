package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/gorilla/websocket"
)

// var DEBUG = false

// func debug(v ...any) {
// 	if DEBUG {
// 		log.Println(v...)
// 	}
// }

var HandshakeTimeout time.Duration = time.Second * 2

type Meta[T any] struct {
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	Index   string `json:"index"`
	Data    T      `json:"data"`
}
type OnMessageCallback[T any] func([]Meta[T])

func Subscribe[T any](ctx context.Context, protocol, host, path string, callback OnMessageCallback[T]) {
	retryCount := 0
	cache := ""
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"
	closingTime := atomic.Bool{}
	wsURL := url.URL{Scheme: protocol, Host: host, Path: path}
	muWsClient := sync.Mutex{}
	var wsClient *websocket.Conn
	_handShakeTimeout := HandshakeTimeout

	go func(ct *atomic.Bool) {
		<-ctx.Done()
		ct.Swap(true)
		muWsClient.Lock()
		defer muWsClient.Unlock()
		if wsClient == nil {
			// debug("subscribe client closing but no connection to close", host, path, ctx.Err())
			return
		}
		// debug("subscribe client closing", host, path, ctx.Err())
		wsClient.Close()
	}(&closingTime)

	for {
		var err error
		quickDial := &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: _handShakeTimeout,
		}

		muWsClient.Lock()
		wsClient, _, err = quickDial.Dial(wsURL.String(), nil)
		if wsClient == nil || err != nil {
			muWsClient.Unlock()
			// debug("failed websocket dial ", err)
			time.Sleep(2 * time.Second)
			continue
		}
		muWsClient.Unlock()
		// debug("client connection stablished", host, path)

		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil || message == nil {
				// debug("failed websocket read connection ", err)
				wsClient.Close()
				break
			}

			result := []Meta[T]{}
			if isList {
				var objs []objects.Object
				cache, objs, err = messages.PatchList(message, cache)
				if err != nil {
					// debug("fail to parse message from websocket", err)
					break
				}
				for _, obj := range objs {
					var item T
					err = json.Unmarshal([]byte(obj.Data), &item)
					if err != nil {
						// debug("fail to unmarshal data from websocket", err)
						continue
					}
					result = append(result, Meta[T]{
						Created: obj.Created,
						Updated: obj.Updated,
						Index:   obj.Index,
						Data:    item,
					})
				}
				retryCount = 0
				callback(result)
				continue
			}

			var obj objects.Object
			cache, obj, err = messages.Patch(message, cache)
			if err != nil {
				// debug("fail to parse message from websocket", err)
				break
			}

			var item T
			err = json.Unmarshal([]byte(obj.Data), &item)
			if err != nil {
				// debug("fail to unmarshal data from websocket", err)
				break
			}
			result = append(result, Meta[T]{
				Created: obj.Created,
				Updated: obj.Updated,
				Index:   obj.Index,
				Data:    item,
			})
			retryCount = 0
			callback(result)
		}

		bye := closingTime.Load()
		if bye {
			// debug("skip reconnection, client closing...", host, path)
			break
		}

		retryCount++
		if retryCount < 30 {
			// debug("reconnecting websocket...", host, path, err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		if retryCount < 100 {
			// debug("reconnecting websocket in 2 seconds...", host, path, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// debug("reconnecting websocket in 10 seconds...", err)
		time.Sleep(10 * time.Second)
	}
}
