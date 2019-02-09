package samo

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestAudit(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Audit = func(r *http.Request) bool {
		return r.Header.Get("Upgrade") != "websocket" && r.Method != "GET" && r.Method != "DELETE"
	}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)

	index, err := app.Storage.Set("test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)

	req := httptest.NewRequest("GET", "/r/sa/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/r/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.console.Err(err)
	require.Error(t, err)

	app.Audit = func(r *http.Request) bool {
		return r.Method == "GET"
	}

	var jsonStr = []byte(`{"data":"test"}`)
	req = httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("GET", "/r/sa/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)

}

func TestDoubleShutdown(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.separator = ":"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	app.Close(os.Interrupt)
}

func TestDoubleStart(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.separator = ":"
	app.Start("localhost:9889")
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
}

func TestRestart(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.separator = ":"
	app.Start("localhost:9889")
	app.Close(os.Interrupt)
	// https://golang.org/pkg/net/http/#example_Server_Shutdown
	app2 := Server{}
	app2.Silence = true
	app2.Start("localhost:9889")
	defer app2.Close(os.Interrupt)
}

func TestInvalidKey(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.separator = ":"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/:test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.console.Err(err)
	require.Error(t, err)
	u = url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test::1"}
	c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.console.Err(err)
	require.Error(t, err)
	u = url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test/1:"}
	c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.console.Err(err)
	require.Error(t, err)

	req := httptest.NewRequest("GET", "/r/sa/test::1", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/r/test::1", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 400, resp.StatusCode)
}
