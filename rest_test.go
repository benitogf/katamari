package katamari_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/storages/level"
	"github.com/stretchr/testify/require"
)

func TestRestPostNonObject(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`non object`)
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRestPostEmptyData(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`{"data":""}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRestPostKey(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`{"data":"test"}`)
	req := httptest.NewRequest("POST", "/test//a", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
}

func TestRestDel(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)

	req := httptest.NewRequest("DELETE", "/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	data, _ := app.Storage.Get("test")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Empty(t, data)

	req = httptest.NewRequest("DELETE", "/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	index, err = app.Storage.Set("test/1", "test1")
	require.NoError(t, err)
	require.Equal(t, "1", index)
	index, err = app.Storage.Set("test/2", "test2")
	require.NoError(t, err)
	require.Equal(t, "2", index)

	req = httptest.NewRequest("DELETE", "/test/*", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	_, err = app.Storage.Get("test/1")
	require.Error(t, err)
	_, err = app.Storage.Get("test/2")
	require.Error(t, err)
}

func TestRestGet(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.InMemoryKeys = []string{"sources"}
	app.Storage = &level.Storage{Path: "test/db"}
	app.Start("localhost:0")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)
	index, err = app.Storage.MemSet("sources", "list")
	require.NoError(t, err)
	require.Equal(t, "sources", index)
	data, _ := app.Storage.Get("test")
	dataSources, _ := app.Storage.MemGet("sources")

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, string(data), string(body))

	req = httptest.NewRequest("GET", "/sources", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, string(dataSources), string(body))

	req = httptest.NewRequest("GET", "/test/notest", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRestStats(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	index, err := app.Storage.Set("test/1", "test1")
	require.NoError(t, err)
	require.NotEmpty(t, index)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"keys\":[\"test/1\"]}", string(body))

	_ = app.Storage.Del("test/1")

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"keys\":[]}", string(body))
}

func TestRestResponseCode(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)

	index, err = app.Storage.Set("test/1", "test1")
	require.NoError(t, err)
	require.NotEmpty(t, index)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req = httptest.NewRequest("GET", "/*", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	req = httptest.NewRequest("DELETE", "/test/1", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRestGetBadRequest(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	req := httptest.NewRequest("GET", "//test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, 301, resp.StatusCode)
}

func TestRestPostInvalidKey(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	req := httptest.NewRequest("POST", "/test/*/*", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRestGetInvalidKey(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	req := httptest.NewRequest("GET", "/test/*/**", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRestDeleteInvalidKey(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	req := httptest.NewRequest("DELETE", "/test/*/**", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
