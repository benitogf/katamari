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

func TestArchetype(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Archetypes = Archetypes{
		"test1": func(index string, data string) bool {
			return data == "test1"
		},
		"test?/*": func(index string, data string) bool {
			return data == "test"
		},
	}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	require.False(t, app.helpers.checkArchetype("test1", "test1", "notest", app.Static, app.Archetypes))
	require.True(t, app.helpers.checkArchetype("test1", "test1", "test1", app.Static, app.Archetypes))
	require.False(t, app.helpers.checkArchetype("test1/1", "1", "test1", app.Static, app.Archetypes))
	require.False(t, app.helpers.checkArchetype("test0/1", "1", "notest", app.Static, app.Archetypes))
	require.True(t, app.helpers.checkArchetype("test0/1", "1", "test", app.Static, app.Archetypes))

	var jsonStr = []byte(`{"data":"notest"}`)
	req := httptest.NewRequest("POST", "/r/sa/test1", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

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

func TestInvalidKey(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.separator = ":"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	require.NotEmpty(t, app.Server)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/:test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.console.Err(err)
	require.Error(t, err)
	u2 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test::1"}
	c, _, err = websocket.DefaultDialer.Dial(u2.String(), nil)
	require.Nil(t, c)
	app.console.Err(err)
	require.Error(t, err)
	u3 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test/1:"}
	c, _, err = websocket.DefaultDialer.Dial(u3.String(), nil)
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
