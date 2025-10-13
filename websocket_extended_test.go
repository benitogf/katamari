package katamari

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/benitogf/katamari/objects"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestWebSocketSubscriptionEvents(t *testing.T) {
	app := Server{}
	app.Silence = true

	subscribed := ""
	unsubscribed := ""
	var mu sync.Mutex

	app.OnSubscribe = func(key string) error {
		mu.Lock()
		subscribed = key
		mu.Unlock()
		return nil
	}

	app.OnUnsubscribe = func(key string) {
		mu.Lock()
		unsubscribed = key
		mu.Unlock()
	}

	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Connect to websocket
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/testkey"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)

	// Give time for subscription event
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	require.Equal(t, "testkey", subscribed)
	mu.Unlock()

	// Close connection
	c.Close()

	// Give time for unsubscription event
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	require.Equal(t, "testkey", unsubscribed)
	mu.Unlock()
}

func TestWebSocketSubscriptionDenied(t *testing.T) {
	app := Server{}
	app.Silence = true

	app.OnSubscribe = func(key string) error {
		if key == "denied" {
			return errors.New("subscription denied")
		}
		return nil
	}

	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Try to connect to denied key
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/denied"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Error(t, err)
	require.Nil(t, c)
}

func TestWebSocketWithVersion(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Set initial data
	_, err := app.Storage.Set("versiontest", `{"version":1}`)
	require.NoError(t, err)

	// wait for the broadcast of the write to generate a new version
	time.Sleep(100 * time.Millisecond)
	// Get current version
	entry, err := app.fetch("versiontest")
	require.NoError(t, err)

	// log.Println("versiontest", strconv.FormatInt(entry.Version, 16))
	// Connect with matching version (should not receive initial data)
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/versiontest"}
	q := u.Query()
	q.Set("v", strconv.FormatInt(entry.Version, 16))
	u.RawQuery = q.Encode()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer c.Close()

	// Set read deadline to avoid blocking
	c.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	// Should not receive initial message due to version match
	_, _, err = c.ReadMessage()
	require.Error(t, err) // Should timeout
}

func TestWebSocketFilteredRoute(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Static = true

	// Don't add any filters - route should be denied in static mode
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Try to connect to unfiltered route in static mode
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/filtered"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Error(t, err)
	require.Nil(t, c)
}

func TestWebSocketConcurrentConnections(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Set initial data
	_, err := app.Storage.Set("concurrent", `{"test":true}`)
	require.NoError(t, err)

	var wg sync.WaitGroup
	numConnections := 5

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			u := url.URL{Scheme: "ws", Host: app.Address, Path: "/concurrent"}
			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.NoError(t, err)
			defer c.Close()

			// Read initial message
			_, message, err := c.ReadMessage()
			require.NoError(t, err)
			obj, err := objects.Decode([]byte(message))
			require.NoError(t, err)
			require.Contains(t, obj.Data, "test")
		}(i)
	}

	wg.Wait()
}

func TestWebSocketBroadcastUpdate(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Set initial data
	_, err := app.Storage.Set("broadcast", `{"value":1}`)
	require.NoError(t, err)

	// Connect websocket
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/broadcast"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer c.Close()

	// Read initial message
	_, message, err := c.ReadMessage()
	require.NoError(t, err)
	obj, err := objects.Decode([]byte(message))
	require.NoError(t, err)
	require.Contains(t, obj.Data, `:1`)

	// Update data (should trigger broadcast)
	_, err = app.Storage.Set("broadcast", `{"value":2}`)
	require.NoError(t, err)

	// Read broadcast message
	c.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, message, err = c.ReadMessage()
	require.NoError(t, err)
	obj, err = objects.Decode([]byte(message))
	require.NoError(t, err)
	require.Contains(t, obj.Data, `:2`)
}

func TestWebSocketGlobSubscription(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Set data matching pattern
	_, err := app.Storage.Set("items/1", `{"id":1}`)
	require.NoError(t, err)
	_, err = app.Storage.Set("items/2", `{"id":2}`)
	require.NoError(t, err)

	// Connect to glob pattern
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/items/*"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer c.Close()

	// Read initial message (should contain both items)
	c.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, message, err := c.ReadMessage()
	require.NoError(t, err)
	obj, err := objects.Decode([]byte(message))
	require.NoError(t, err)
	require.Contains(t, obj.Data, `:1`)
	require.Contains(t, obj.Data, `:2`)
}

func TestWebSocketReadFilter(t *testing.T) {
	app := Server{}
	app.Silence = true

	// Add read filter that modifies data
	app.ReadFilter("filtered", func(key string, data []byte) ([]byte, error) {
		return []byte(`{"filtered":true}`), nil
	})

	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Set original data
	_, err := app.Storage.Set("filtered", `{"original":true}`)
	require.NoError(t, err)

	// Connect websocket
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/filtered"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer c.Close()

	// Read message (should be filtered)
	c.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, message, err := c.ReadMessage()
	require.NoError(t, err)
	obj, err := objects.Decode([]byte(message))
	require.NoError(t, err)
	require.Contains(t, obj.Data, "filtered")
	require.NotContains(t, obj.Data, "original")
}
