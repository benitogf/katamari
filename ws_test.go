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
	t.Parallel()
	var wg sync.WaitGroup
	app := Server{}
	app.Silence = true
	app.Tick = 100 * time.Millisecond
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/"}
	c1, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	c2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	wg.Add(3)
	go func() {
		for {
			_, message, err := c1.ReadMessage()
			if err != nil {
				app.console.Err("read c1", err)
				break
			}
			app.console.Log("time c1", string(message))
			wg.Done()
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
		wg.Done()
	}

	wg.Wait()

	err = c1.Close()
	require.NoError(t, err)
}
