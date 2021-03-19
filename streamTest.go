package katamari

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/benitogf/jsonpatch"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// StreamBroadcastTest testing stream function
func StreamBroadcastTest(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	var postObject objects.Object
	var wsObject objects.Object
	var wsEvent messages.Message
	var wsCache string
	testData := messages.Encode([]byte("something 🧰"))
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
			wsEvent, err = messages.DecodeTest(message)
			require.NoError(t, err)
			app.console.Log("read wsClient", wsEvent.Data)
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	wsCache = wsEvent.Data
	wsVersion, err := strconv.ParseInt(wsEvent.Version, 16, 64)
	require.NoError(t, err)
	streamCache, err := app.Stream.GetCache("test")
	require.NoError(t, err)
	app.console.Log("post data")
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, wsVersion, streamCache.Version)
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
	var postObject objects.Object
	var wsObject objects.Object
	var wsEvent messages.Message
	var wsCache string
	testData := messages.Encode([]byte("something 🧰"))
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
			wsEvent, err = messages.DecodeTest(message)
			require.NoError(t, err)
			app.console.Log("read wsClient", wsEvent.Data)
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	wsCache = wsEvent.Data
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	req := httptest.NewRequest("POST", "/test/1", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
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
func StreamGlobBroadcastTest(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	var postObject objects.Object
	var wsObject []objects.Object
	var wsEvent messages.Message
	var wsCache string
	testData := messages.Encode([]byte("something 🧰"))
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
			wsEvent, err = messages.DecodeTest(message)
			require.NoError(t, err)
			app.console.Log("read wsClient", wsEvent.Data)
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	wsCache = wsEvent.Data
	app.console.Log("post data")
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	req := httptest.NewRequest("POST", "/test/*", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	wg.Wait()
	wg.Add(1)
	patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
	require.NoError(t, err)
	modified, err := patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)
	wsCache = string(modified)

	require.Equal(t, wsObject[0].Index, postObject.Index)
	require.Equal(t, wsObject[0].Data, testData)

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

	require.Equal(t, len(wsObject), 0)
}

// StreamBroadcastFilterTest testing stream function
func StreamBroadcastFilterTest(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	var postObject objects.Object
	var wsExtraEvent messages.Message
	testData := messages.Encode([]byte("something 🧰"))
	var jsonStr = []byte(`{"data":"` + testData + `"}`)
	// extra filter
	app.ReadFilter("extra", func(index string, data []byte) ([]byte, error) {
		return []byte("extra"), nil
	})
	// extra route
	app.Router = mux.NewRouter()
	app.Router.HandleFunc("/extra", func(w http.ResponseWriter, r *http.Request) {
		client, err := app.Stream.New("test/*", "extra", w, r)
		if err != nil {
			return
		}

		entry, err := app.Fetch("test/*", "extra")
		if err != nil {
			return
		}

		go app.Stream.Write(client, messages.Encode(entry.Data), true, entry.Version)
		app.Stream.Read("test/*", "extra", client)
	}).Methods("GET")
	app.Start("localhost:0")
	app.Storage.Clear()
	wsExtraURL := url.URL{Scheme: "ws", Host: app.Address, Path: "/extra"}
	wsExtraClient, _, err := websocket.DefaultDialer.Dial(wsExtraURL.String(), nil)
	require.NoError(t, err)
	wg.Add(1)
	go func() {
		for {
			_, message, err := wsExtraClient.ReadMessage()
			if err != nil {
				break
			}
			wsExtraEvent, err = messages.DecodeTest(message)
			require.NoError(t, err)
			app.console.Log("read wsClient", wsExtraEvent.Data)
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(1)
	app.console.Log("post data")
	req := httptest.NewRequest("POST", "/test/*", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &postObject)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	wg.Wait()
	wsExtraClient.Close()

	require.Equal(t, "extra", wsExtraEvent.Data)
}
