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
	app.ReceiveFilter("test1", func(index string, data []byte) ([]byte, error) {
		app.console.Log(string(data) != "test1")
		if string(data) != "test1" {
			return nil, errors.New("filtered")
		}

		return data, nil
	})
	app.ReceiveFilter("test?/*", func(index string, data []byte) ([]byte, error) {
		if string(data) != "test" {
			return nil, errors.New("filtered")
		}

		return data, nil
	})
	app.SendFilter("bag/*", func(index string, data []byte) ([]byte, error) {
		return []byte("intercepted"), nil
	})
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	_, err := app.Filters.Receive.check("test1", "test1", []byte("notest"), false)
	require.Error(t, err)
	_, err = app.Filters.Receive.check("test1", "test1", []byte("test1"), false)
	require.NoError(t, err)
	data, err := app.Filters.Send.check("bag/1", "1", []byte("test"), false)
	require.NoError(t, err)
	require.Equal(t, "intercepted", string(data))

	var jsonStr = []byte(`{"data":"notest"}`)
	req := httptest.NewRequest("POST", "/r/sa/test1", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)

	req = httptest.NewRequest("POST", "/r/sa/bag/1", bytes.NewBuffer(jsonStr))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)

	req = httptest.NewRequest("GET", "/r/sa/bag/1", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "intercepted", string(body))
}
