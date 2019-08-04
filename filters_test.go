package samo

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilters(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.WriteFilter("test1", func(key string, data []byte) ([]byte, error) {
		app.console.Log(string(data) != "test1")
		if string(data) != "test1" {
			return nil, errors.New("filtered")
		}

		return data, nil
	})
	app.WriteFilter("test?/*", func(key string, data []byte) ([]byte, error) {
		if string(data) != "test" {
			return nil, errors.New("filtered")
		}

		return data, nil
	})
	app.ReadFilter("bag/*", func(key string, data []byte) ([]byte, error) {
		return []byte("intercepted:" + key), nil
	})
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	_, err := app.Filters.Write.check("test1", []byte("notest"), false)
	require.Error(t, err)
	_, err = app.Filters.Write.check("test1", []byte("test1"), false)
	require.NoError(t, err)
	data, err := app.Filters.Read.check("bag/1", []byte("test"), false)
	require.NoError(t, err)
	require.Equal(t, "intercepted:bag/1", string(data))

	var jsonStr = []byte(`{"data":"` + app.messages.encode([]byte("notest")) + `"}`)
	req := httptest.NewRequest("POST", "/test1", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)

	req = httptest.NewRequest("POST", "/bag/1", bytes.NewBuffer(jsonStr))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)

	req = httptest.NewRequest("GET", "/bag/1", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "intercepted:bag/1", string(body))
}
