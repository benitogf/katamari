package samo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/benitogf/jsonpatch"
	"github.com/benitogf/nsocket"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func streamBroadcast(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var postObject Object
	var wsObject Object
	var nsObject Object
	var wsEvent message
	var nsEvent message
	var wsCache string
	var nsCache string
	testData := app.messages.encode([]byte("something 🧰"))
	wsURL := url.URL{Scheme: "ws", Host: app.address, Path: "/test"}
	wsClient, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	nsClient, err := nsocket.Dial(app.NamedSocket, "test")
	require.NoError(t, err)
	wg.Add(2)
	go func() {
		for {
			message, err := nsClient.Read()
			if err != nil {
				break
			}
			mutex.Lock()
			nsEvent, err = app.messages.decodeTest([]byte(message))
			require.NoError(t, err)
			app.console.Log("read nsClient", nsEvent.Data)
			mutex.Unlock()
			wg.Done()
		}
	}()
	go func() {
		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil {
				break
			}
			mutex.Lock()
			wsEvent, err = app.messages.decodeTest(message)
			require.NoError(t, err)
			app.console.Log("read wsClient", wsEvent.Data)
			mutex.Unlock()
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(2)
	streamCache, err := app.stream.getPoolCache("test")
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
	mutex.Lock()
	wsCache = wsEvent.Data
	nsCache = nsEvent.Data
	wsVersion, err := strconv.ParseInt(wsEvent.Version, 16, 64)
	require.NoError(t, err)
	require.Equal(t, wsVersion, streamCache.version)
	mutex.Unlock()
	wg.Wait()
	wg.Add(2)

	mutex.Lock()
	patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err := patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)
	mutex.Lock()
	wsCache = string(modified)
	mutex.Unlock()

	mutex.Lock()
	patch, err = jsonpatch.DecodePatch([]byte(nsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(nsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &nsObject)
	require.NoError(t, err)
	mutex.Lock()
	nsCache = string(modified)
	mutex.Unlock()

	require.Equal(t, wsObject.Index, postObject.Index)
	require.Equal(t, nsObject.Index, postObject.Index)
	require.Equal(t, wsObject.Data, testData)
	require.Equal(t, nsObject.Data, testData)

	req = httptest.NewRequest("DELETE", "/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	wg.Wait()

	mutex.Lock()
	patch, err = jsonpatch.DecodePatch([]byte(wsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)

	mutex.Lock()
	patch, err = jsonpatch.DecodePatch([]byte(nsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(nsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &nsObject)
	require.NoError(t, err)

	nsClient.Close()
	wsClient.Close()

	require.Equal(t, wsObject.Created, int64(0))
	require.Equal(t, nsObject.Created, int64(0))
}

func streamGlobBroadcast(t *testing.T, app *Server) {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var postObject Object
	var wsObject []Object
	var nsObject []Object
	var wsEvent message
	var nsEvent message
	var wsCache string
	var nsCache string
	testData := app.messages.encode([]byte("something 🧰"))
	wsURL := url.URL{Scheme: "ws", Host: app.address, Path: "/test/*"}
	wsClient, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	require.NoError(t, err)
	nsClient, err := nsocket.Dial(app.NamedSocket, "test/*")
	require.NoError(t, err)
	wg.Add(2)
	go func() {
		for {
			message, err := nsClient.Read()
			if err != nil {
				break
			}
			mutex.Lock()
			nsEvent, err = app.messages.decodeTest([]byte(message))
			require.NoError(t, err)
			app.console.Log("read nsClient", nsEvent.Data)
			mutex.Unlock()
			wg.Done()
		}
	}()
	go func() {
		for {
			_, message, err := wsClient.ReadMessage()
			if err != nil {
				break
			}
			mutex.Lock()
			wsEvent, err = app.messages.decodeTest(message)
			require.NoError(t, err)
			app.console.Log("read wsClient", wsEvent.Data)
			mutex.Unlock()
			wg.Done()
		}
	}()
	wg.Wait()
	wg.Add(2)
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
	mutex.Lock()
	wsCache = wsEvent.Data
	nsCache = nsEvent.Data
	mutex.Unlock()
	wg.Wait()
	wg.Add(2)

	mutex.Lock()
	patch, err := jsonpatch.DecodePatch([]byte(wsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err := patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)
	mutex.Lock()
	wsCache = string(modified)
	mutex.Unlock()

	mutex.Lock()
	patch, err = jsonpatch.DecodePatch([]byte(nsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(nsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &nsObject)
	require.NoError(t, err)
	mutex.Lock()
	nsCache = string(modified)
	mutex.Unlock()

	require.Equal(t, wsObject[0].Index, postObject.Index)
	require.Equal(t, nsObject[0].Index, postObject.Index)
	require.Equal(t, wsObject[0].Data, testData)
	require.Equal(t, nsObject[0].Data, testData)

	req = httptest.NewRequest("DELETE", "/test/*", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	wg.Wait()

	mutex.Lock()
	patch, err = jsonpatch.DecodePatch([]byte(wsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(wsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &wsObject)
	require.NoError(t, err)

	mutex.Lock()
	patch, err = jsonpatch.DecodePatch([]byte(nsEvent.Data))
	mutex.Unlock()
	require.NoError(t, err)
	modified, err = patch.Apply([]byte(nsCache))
	require.NoError(t, err)
	err = json.Unmarshal(modified, &nsObject)
	require.NoError(t, err)

	nsClient.Close()
	wsClient.Close()

	require.Equal(t, len(wsObject), 0)
	require.Equal(t, len(nsObject), 0)
}

func TestStreamBroadcastMemory(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.forcePatch = true
	app.NamedSocket = "samotest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	streamBroadcast(t, &app)
}

func TestStreamBroadcastLevel(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.forcePatch = true
	app.NamedSocket = "samotest"
	app.Storage = &LevelStorage{
		Path: "test/db"}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	streamBroadcast(t, &app)
}

func TestStreamGlobBroadcastMemory(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.forcePatch = true
	app.NamedSocket = "samotest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	streamGlobBroadcast(t, &app)
}

func TestStreamGlobBroadcastLevel(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.forcePatch = true
	app.NamedSocket = "samotest"
	app.Storage = &LevelStorage{
		Path: "test/db"}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	streamGlobBroadcast(t, &app)
}
