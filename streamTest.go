package katamari

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/goccy/go-json"

	"github.com/benitogf/jsonpatch"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/expect"
	"github.com/stretchr/testify/require"
)

// StreamBroadcastTest testing stream function
func StreamBroadcastTest(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	// this lock should not be neccesary but the race detector doesnt recognize the wait group preventing the race here
	var lk sync.Mutex
	var postObject objects.Object
	var wsObject objects.Object
	var wsEvent messages.Message
	var wsCache string
	testData := messages.Encode([]byte(TEST_DATA))
	wsURL := url.URL{Scheme: "ws", Host: app.Address, Path: "/test"}
	wsClient, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil {
				break
			}
			lk.Lock()
			wsEvent, err = messages.DecodeBuffer(message)
			lk.Unlock()
			expect.Nil(err)
			app.Console.Log("read wsClient", wsEvent.Data)
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	lk.Lock()
	wsCache = wsEvent.Data
	wsVersion, err := strconv.ParseInt(wsEvent.Version, 16, 64)
	lk.Unlock()
	require.NoError(t, err)
	streamCacheVersion, err := app.Stream.GetCacheVersion("test")
	require.NoError(t, err)
	app.Console.Log("post data")
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, wsVersion, streamCacheVersion)
	wg.Wait()
	wg.Add(1)

	patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
	require.NoError(t, err)
	modified, err := patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)

	require.Equal(t, wsObject.Index, postObject.Index)
	require.Equal(t, wsObject.Data, testData)

	req = httptest.NewRequest("DELETE", "/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	wg.Wait()

	patch, err = jsonpatch.DecodePatch([]byte(wsEvent.Data))
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)

	wsClient.Close()

	require.Equal(t, wsObject.Created, int64(0))
}

// StreamItemGlobBroadcastTest testing stream function
func StreamItemGlobBroadcastTest(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	// this lock should not be neccesary but the race detector doesnt recognize the wait group preventing the race here
	var lk sync.Mutex
	var postObject objects.Object
	var wsObject objects.Object
	var wsEvent messages.Message
	var wsCache string
	testData := messages.Encode([]byte(TEST_DATA))
	wsURL := url.URL{Scheme: "ws", Host: app.Address, Path: "/test/1"}
	wsClient, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil {
				break
			}
			lk.Lock()
			wsEvent, err = messages.DecodeBuffer(message)
			lk.Unlock()
			expect.Nil(err)
			app.Console.Log("read wsClient", wsEvent.Data)
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	lk.Lock()
	wsCache = wsEvent.Data
	lk.Unlock()
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	req := httptest.NewRequest("POST", "/test/1", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	wg.Wait()
	patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
	require.NoError(t, err)
	modified, err := patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)
	wsCache = string(modified)

	require.Equal(t, wsObject.Index, postObject.Index)
	require.Equal(t, wsObject.Data, testData)

	wg.Add(1)
	req = httptest.NewRequest("DELETE", "/test/*", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	wg.Wait()

	patch, err = jsonpatch.DecodePatch([]byte(wsEvent.Data))
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)

	wsClient.Close()

	require.Equal(t, int64(0), wsObject.Created)
}

// StreamGlobBroadcastTest testing stream function
func StreamGlobBroadcastTest(t *testing.T, app *Server, n int) {
	var wg sync.WaitGroup
	// this lock should not be neccesary but the race detector doesnt recognize the wait group preventing the race here
	var lk sync.Mutex
	var postObject objects.Object
	var wsObject []objects.Object
	var wsEvent messages.Message
	var wsCache string
	testData := messages.Encode([]byte(TEST_DATA))
	wsURL := url.URL{Scheme: "ws", Host: app.Address, Path: "/test/*"}
	wsClient, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil {
				break
			}
			lk.Lock()
			wsEvent, err = messages.DecodeBuffer(message)
			lk.Unlock()
			expect.Nil(err)
			app.Console.Log("read wsClient", wsEvent.Data)
			wg.Done()
		}
	}()
	wg.Wait()

	lk.Lock()
	wsCache = wsEvent.Data
	lk.Unlock()
	app.Console.Log("post data")
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	for i := 0; i < n; i++ {
		wg.Add(1)
		req := httptest.NewRequest("POST", "/test/*", bytes.NewBuffer(jsonStr))
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(body, &postObject)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		wg.Wait()
		lk.Lock()
		if !wsEvent.Snapshot {
			patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
			require.NoError(t, err)
			modified, err := patch.Apply([]byte(wsCache))
			require.NoError(t, err)
			err = json.Unmarshal(modified, &wsObject)
			require.NoError(t, err)
			wsCache = string(modified)
		} else {
			wsCache = wsEvent.Data
		}
		lk.Unlock()
	}

	require.Equal(t, wsObject[0].Index, postObject.Index)
	require.Equal(t, wsObject[0].Data, testData)

	wg.Add(1)
	req := httptest.NewRequest("DELETE", "/test/*", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	wg.Wait()

	lk.Lock()
	if !wsEvent.Snapshot {
		patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
		require.NoError(t, err)
		modified, err := patch.Apply([]byte(wsCache))
		require.NoError(t, err)
		err = json.Unmarshal(modified, &wsObject)
		require.NoError(t, err)
	} else {
		err = json.Unmarshal([]byte(wsEvent.Data), &wsObject)
		require.NoError(t, err)
	}
	lk.Unlock()

	wsClient.Close()

	require.Equal(t, len(wsObject), 0)
}

// StreamBroadcastFilterTest testing stream function
func StreamBroadcastFilterTest(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	var postObject objects.Object
	var wsExtraEvent messages.Message
	testData := messages.Encode([]byte(TEST_DATA))
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	// extra filter
	app.ReadFilter("test/*", func(index string, data []byte) ([]byte, error) {
		return []byte("extra"), nil
	})
	// extra route
	app.Router = mux.NewRouter()
	app.Start("localhost:0")
	app.Storage.Clear()
	wsExtraURL := url.URL{Scheme: "ws", Host: app.Address, Path: "/test/*"}
	wsExtraClient, _, err := websocket.DefaultDialer.Dial(wsExtraURL.String(), nil)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		for {
			_, message, err := wsExtraClient.ReadMessage()
			if err != nil {
				break
			}
			wsExtraEvent, err = messages.DecodeBuffer(message)
			expect.Nil(err)
			app.Console.Log("read wsClient", string(message))
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	app.Console.Log("post data")
	req := httptest.NewRequest("POST", "/test/*", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	wg.Wait()
	wsExtraClient.Close()

	// empty operations for a broadcast with no changes
	require.Equal(t, false, wsExtraEvent.Snapshot)
	require.Equal(t, "[]", wsExtraEvent.Data)
}

// StreamBroadcastNoPatchTest testing stream function
func StreamBroadcastNoPatchTest(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	var postObject objects.Object
	var wsExtraEvent messages.Message

	testData := messages.Encode([]byte(TEST_DATA))
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	// extra route
	app.Router = mux.NewRouter()
	app.NoPatch = true
	app.Start("localhost:0")
	app.Storage.Clear()
	wsExtraURL := url.URL{Scheme: "ws", Host: app.Address, Path: "/test/*"}
	wsExtraClient, _, err := websocket.DefaultDialer.Dial(wsExtraURL.String(), nil)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		for {
			_, message, err := wsExtraClient.ReadMessage()
			if err != nil {
				break
			}
			wsExtraEvent, err = messages.DecodeBuffer(message)
			expect.Nil(err)
			expect.True(wsExtraEvent.Snapshot)
			app.Console.Log("read wsClient", string(message))
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	app.Console.Log("post data")
	req := httptest.NewRequest("POST", "/test/*", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	wg.Wait()

	wg.Add(1)
	app.Console.Log("post data")
	req = httptest.NewRequest("POST", "/test/*", bytes.NewBuffer(jsonStr))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	wg.Wait()

	wsExtraClient.Close()
}
