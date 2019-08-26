package katamari

import (
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestWsTime(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Tick = 100 * time.Millisecond
	mutex := sync.Mutex{}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/"}
	c1, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	c2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	count := 0

	go func() {
		for {
			_, message, err := c1.ReadMessage()
			if err != nil {
				app.console.Err("read c1", err)
				break
			}
			app.console.Log("time c1", string(message))
			mutex.Lock()
			count++
			mutex.Unlock()
		}
	}()

	for {
		_, message, err := c2.ReadMessage()
		if err != nil {
			app.console.Err("read c2", err)
			break
		}
		app.console.Log("time c2", string(message))
		err = c2.Close()
		require.NoError(t, err)
	}

	tryes := 0
	mutex.Lock()
	for count < 2 && tryes < 10000 {
		tryes++
		mutex.Unlock()
		time.Sleep(1 * time.Millisecond)
		mutex.Lock()
	}
	mutex.Unlock()

	err = c1.Close()
	require.NoError(t, err)
}
