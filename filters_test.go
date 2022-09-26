package katamari

import (
	"bytes"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/benitogf/jsondiff"
	"github.com/goccy/go-json"

	"github.com/stretchr/testify/require"
)

func TestFilters(t *testing.T) {
	app := Server{}
	app.Silence = true
	unacceptedData := json.RawMessage(`{"test":false}`)
	acceptedData := json.RawMessage(`{"test":true}`)
	uninterceptedData := json.RawMessage(`{"intercepted":false}`)
	interceptedData := json.RawMessage(`{"intercepted":true}`)
	filteredData := json.RawMessage(`{"filtered":true}`)
	unfilteredData := json.RawMessage(`{"filtered":false}`)
	notified := false
	app.WriteFilter("test1", func(key string, data json.RawMessage) (json.RawMessage, error) {
		comparison, _ := jsondiff.Compare(data, unfilteredData, &jsondiff.Options{})
		if comparison != jsondiff.FullMatch {
			return nil, errors.New("filtered")
		}

		return data, nil
	})
	app.WriteFilter("test/*", func(key string, data json.RawMessage) (json.RawMessage, error) {
		comparison, _ := jsondiff.Compare(data, acceptedData, &jsondiff.Options{})
		if comparison != jsondiff.FullMatch {
			return nil, errors.New("filtered")
		}

		return data, nil
	})
	app.ReadFilter("bag/*", func(key string, data json.RawMessage) (json.RawMessage, error) {
		return interceptedData, nil
	})

	app.WriteFilter("book/*", func(key string, data json.RawMessage) (json.RawMessage, error) {
		return data, nil
	})
	app.ReadFilter("book/*", func(key string, data json.RawMessage) (json.RawMessage, error) {
		return data, nil
	})
	app.AfterWrite("flyer", func(key string) {
		notified = true
	})

	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	_, err := app.filters.Write.check("test/1", unacceptedData, false)
	require.Error(t, err)
	_, err = app.filters.Write.check("test/1", acceptedData, false)
	require.NoError(t, err)
	data, err := app.filters.Read.check("bag/1", uninterceptedData, false)
	require.NoError(t, err)
	comparison, _ := jsondiff.Compare(data, interceptedData, &jsondiff.Options{})
	require.Equal(t, comparison, jsondiff.FullMatch)
	_, err = app.filters.Write.check("test1", filteredData, false)
	require.Error(t, err)
	_, err = app.filters.Write.check("test1", unfilteredData, false)
	require.NoError(t, err)
	// test static
	_, err = app.filters.Write.check("book", unacceptedData, true)
	require.Error(t, err)
	_, err = app.filters.Write.check("book/1/1", unacceptedData, true)
	require.Error(t, err)
	_, err = app.filters.Write.check("book/1/1/1", unacceptedData, true)
	require.Error(t, err)
	_, err = app.filters.Read.check("book", unacceptedData, true)
	require.Error(t, err)
	_, err = app.filters.Read.check("book/1/1", unacceptedData, true)
	require.Error(t, err)
	_, err = app.filters.Read.check("book/1/1/1", unacceptedData, true)
	require.Error(t, err)
	_, err = app.filters.Write.check("book/1", unfilteredData, true)
	require.NoError(t, err)
	_, err = app.filters.Read.check("book/1", unfilteredData, true)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/test/1", bytes.NewBuffer(TEST_DATA))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)

	req = httptest.NewRequest("POST", "/bag/1", bytes.NewBuffer(uninterceptedData))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)

	req = httptest.NewRequest("POST", "/flyer", bytes.NewBuffer(TEST_DATA))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)
	require.True(t, notified)

	req = httptest.NewRequest("GET", "/bag/1", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	comparison, _ = jsondiff.Compare(body, interceptedData, &jsondiff.Options{})
	require.Equal(t, comparison, jsondiff.FullMatch)
}
