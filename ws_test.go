package samo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/benitogf/jsonpatch"
	"github.com/gorilla/websocket"
	"github.com/nsf/jsondiff"
	"github.com/stretchr/testify/require"
)

func TestWsTime(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Tick = 100 * time.Millisecond
	mutex := sync.Mutex{}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/time"}
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

func wsRestBroadcast(t *testing.T, app *Server) {
	mutex := sync.RWMutex{}
	_ = app.Storage.Del("test")
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	started := false
	got := ""

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				app.console.Err("read c", err)
				break
			}
			event, err := app.messages.decode(message)
			require.NoError(t, err)
			app.console.Log("read c", event.Data)
			mutex.Lock()
			if started {
				got = event.Data
				err = c.Close()
				require.NoError(t, err)
			}
			started = true
			mutex.Unlock()
		}
	}()

	tryes := 0
	mutex.RLock()
	for !started && tryes < 10000 {
		tryes++
		mutex.RUnlock()
		time.Sleep(1 * time.Millisecond)
		mutex.RLock()
	}
	mutex.RUnlock()
	var jsonStr = []byte(`{"data":"` + app.messages.encode([]byte("Buy coffee and bread for breakfast.")) + `"}`)
	req := httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	tryes = 0
	mutex.RLock()
	for got == "" && tryes < 10000 {
		tryes++
		mutex.RUnlock()
		time.Sleep(1 * time.Millisecond)
		mutex.RLock()
	}
	mutex.RUnlock()
	var wsObject Object
	err = json.Unmarshal([]byte(got), &wsObject)
	require.NoError(t, err)
	var rPostObject Object
	err = json.Unmarshal(body, &rPostObject)
	require.NoError(t, err)
	require.Equal(t, wsObject.Index, rPostObject.Index)
	require.Equal(t, 200, resp.StatusCode)
}

func TestWsRestBroadcastMemory(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	wsRestBroadcast(t, &app)
}

func TestWsRestBroadcastLevel(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Storage = &LevelStorage{
		Path: "test/db"}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	wsRestBroadcast(t, &app)
}

func TestWsRestBroadcastEtcd(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Storage = &EtcdStorage{OnlyClient: true}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	wsRestBroadcast(t, &app)
}

func wsBroadcast(t *testing.T, app *Server) {
	mutex := sync.RWMutex{}
	index, err := app.Storage.Set("test/1", "1", time.Now().UTC().UnixNano(), app.messages.encode([]byte("test")))
	require.NoError(t, err)
	require.Equal(t, "1", index)

	index, err = app.Storage.Set("test/2", "2", time.Now().UTC().UnixNano(), app.messages.encode([]byte("test")))
	require.NoError(t, err)
	require.Equal(t, "2", index)

	u1 := url.URL{Scheme: "ws", Host: app.address, Path: "/mo/test"}
	u2 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test/1"}
	c1, _, err := websocket.DefaultDialer.Dial(u1.String(), nil)
	require.NoError(t, err)
	c2, _, err := websocket.DefaultDialer.Dial(u2.String(), nil)
	require.NoError(t, err)
	wrote := false
	readCount := 0
	got1 := ""
	got2 := ""
	cache1 := ""
	cache2 := ""

	go func() {
		for {
			_, message, err := c1.ReadMessage()
			if err != nil {
				app.console.Err("read c1", err)
				break
			}
			event, err := app.messages.decode(message)
			require.NoError(t, err)
			app.console.Log("read c1", event.Data)
			mutex.Lock()
			if readCount == 2 {
				got1 = event.Data
				err = c1.Close()
				require.NoError(t, err)
			}
			if readCount == 1 {
				patch, err := jsonpatch.DecodePatch([]byte(event.Data))
				require.NoError(t, err)

				modified, err := patch.Apply([]byte(cache1))
				require.NoError(t, err)

				cache1 = string(modified)
			}
			if readCount == 0 {
				cache1 = event.Data
				req := httptest.NewRequest("DELETE", "/r/test/2", nil)
				w := httptest.NewRecorder()
				app.Router.ServeHTTP(w, req)
				resp := w.Result()
				require.Equal(t, http.StatusNoContent, resp.StatusCode)
			}
			readCount++
			mutex.Unlock()
		}
	}()

	tryes := 0
	mutex.RLock()
	for readCount < 2 && tryes < 10000 {
		tryes++
		mutex.RUnlock()
		time.Sleep(1 * time.Millisecond)
		mutex.RLock()
	}
	mutex.RUnlock()

	for {
		_, message, err := c2.ReadMessage()
		if err != nil {
			app.console.Err("read", err)
			break
		}
		event, err := app.messages.decode(message)
		require.NoError(t, err)
		app.console.Log("read c2", event.Data)
		mutex.Lock()
		if wrote {
			got2 = event.Data
			err = c2.Close()
			require.NoError(t, err)
		} else {
			cache2 = event.Data
			app.console.Log("writing from c2")
			req := httptest.NewRequest("POST", "/r/sa/test/1", bytes.NewBuffer([]byte("{"+
				"\"index\": \"1\","+
				"\"data\": \""+app.messages.encode([]byte("test2"))+"\""+
				"}")))
			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)
			resp := w.Result()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			wrote = true
		}
		mutex.Unlock()
	}

	tryes = 0
	mutex.RLock()
	for got1 == "" && tryes < 10000 {
		tryes++
		mutex.RUnlock()
		time.Sleep(1 * time.Millisecond)
		mutex.RLock()
	}
	mutex.RUnlock()

	patch1, err := jsonpatch.DecodePatch([]byte(got1))
	require.NoError(t, err)

	modified1, err := patch1.Apply([]byte(cache1))
	require.NoError(t, err)

	patch2, err := jsonpatch.DecodePatch([]byte(got2))
	require.NoError(t, err)

	modified2, err := patch2.Apply([]byte(cache2))
	require.NoError(t, err)

	opts := jsondiff.DefaultConsoleOptions()
	result, _ := jsondiff.Compare(
		modified1,
		[]byte("["+string(modified2)+"]"),
		&opts)

	app.console.Log("patches: ", got1, got2)
	app.console.Log("merged: ", string(modified1), "["+string(modified2)+"]")

	require.Equal(t, result, jsondiff.FullMatch)
}

func TestWsBroadcastMemory(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	wsBroadcast(t, &app)
}

func TestWsBroadcastLevel(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Storage = &LevelStorage{
		Path: "test/db"}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	wsBroadcast(t, &app)
}

func TestWsBroadcastEtcd(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Storage = &EtcdStorage{OnlyClient: true}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	wsBroadcast(t, &app)
}

func TestWsBadRequest(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	req := httptest.NewRequest("GET", "/sa/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, 400, resp.StatusCode)
}
