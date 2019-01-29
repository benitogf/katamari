package samo

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRestPostNonObject(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`non object`)
	req := httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestRestPostEmptyData(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`{"data":""}`)
	req := httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestRestPostKey(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.separator = ":"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`{"data":"test"}`)
	req := httptest.NewRequest("POST", "/r/sa/test::a", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestRestDel(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)

	req := httptest.NewRequest("DELETE", "/r/test", nil)
	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp := w.Result()
	data, _ := app.Storage.Get("sa", "test")
	require.Equal(t, 204, resp.StatusCode)
	require.Empty(t, data)

	req = httptest.NewRequest("DELETE", "/r/test", nil)
	w = httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 404, resp.StatusCode)
}

func TestRestGet(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)
	data, _ := app.Storage.Get("sa", "test")

	req := httptest.NewRequest("GET", "/r/sa/test", nil)
	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, string(data), string(body))

	req = httptest.NewRequest("GET", "/r/sa/test/notest", nil)
	w = httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 404, resp.StatusCode)
}

func TestRestStats(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)

	index, err := app.Storage.Set("test/1", "1", time.Now().UTC().UnixNano(), "test1")
	require.NoError(t, err)
	require.NotEmpty(t, index)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"keys\":[\"test/1\"]}", string(body))

	_ = app.Storage.Del("test/1")

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"keys\":[]}", string(body))
}

func TestRestResponseCode(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)

	index, err := app.Storage.Set("test", "test", 0, "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)

	index, err = app.Storage.Set("test", "test", 0, "test0")
	require.NoError(t, err)
	require.NotEmpty(t, index)

	index, err = app.Storage.Set("test/1", "1", 0, "test1")
	require.NoError(t, err)
	require.NotEmpty(t, index)

	req := httptest.NewRequest("GET", "/r/sa/test", nil)
	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode)

	req = httptest.NewRequest("GET", "/r/mo/test", nil)
	w = httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/r/test", nil)
	w = httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 204, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/r/test/1", nil)
	w = httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 204, resp.StatusCode)

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)
}
