package katamari

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestAudit(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.Audit = func(r *http.Request) bool {
		return r.Header.Get("Upgrade") != "websocket" && r.Method != "GET" && r.Method != "DELETE"
	}
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/sa/test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.Console.Err(err)
	require.Error(t, err)

	app.Audit = func(r *http.Request) bool {
		return r.Method == "GET"
	}

	var jsonStr = []byte(`{"data":"test"}`)
	req = httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonStr))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)

	app.Audit = func(r *http.Request) bool {
		return r.Header.Get("Upgrade") != "websocket"
	}

	u = url.URL{Scheme: "ws", Host: app.Address, Path: "/"}
	c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.Console.Err(err)
	require.Error(t, err)
}

func TestDoubleShutdown(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	app.Close(os.Interrupt)
}

func TestDoubleStart(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
}

func TestRestart(t *testing.T) {
	t.Skip()
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	app.Close(os.Interrupt)
	// https://golang.org/pkg/net/http/#example_Server_Shutdown
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
}

func TestGlobKey(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/ws/test/*"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	app.Console.Err(err)
	require.NotNil(t, c)
	require.NoError(t, err)
	c.Close()
}

func TestInvalidKey(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.Address, Path: "/sa//test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.Console.Err(err)
	require.Error(t, err)
	u = url.URL{Scheme: "ws", Host: app.Address, Path: "/sa/test//1"}
	c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.Console.Err(err)
	require.Error(t, err)
	u = url.URL{Scheme: "ws", Host: app.Address, Path: "/sa/test/1/"}
	c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.Console.Err(err)
	require.Error(t, err)

	req := httptest.NewRequest("GET", "/test//1", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusMovedPermanently, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/r/test//1", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
}

func TestDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		// TODO: investigate how to simulate a delay in the request on windows
		t.Skip()
	}
	app := Server{
		Deadline: 1 * time.Nanosecond,
		Silence:  true,
	}
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	var jsonStr = []byte(`{"data":"test"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 503, resp.StatusCode)
}
