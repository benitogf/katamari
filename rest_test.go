package katamari_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/objects"
	"github.com/goccy/go-json"
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
	app.Start("localhost:0")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)
	index, err = app.Storage.Set("sources", "list")
	require.NoError(t, err)
	require.Equal(t, "sources", index)
	data, _ := app.Storage.Get("test")
	dataSources, _ := app.Storage.Get("sources")

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, string(data), string(body))

	req = httptest.NewRequest("GET", "/sources", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = io.ReadAll(resp.Body)
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
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"keys\":[\"test/1\"]}", string(body))

	_ = app.Storage.Del("test/1")

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = io.ReadAll(resp.Body)
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

func TestRestPatch(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// {"field1":"value1","field2":"value2"} base64 encoded
	var jsonStr = []byte(`{"data":"eyJmaWVsZDEiOiJ2YWx1ZTEiLCJmaWVsZDIiOiJ2YWx1ZTIifQ=="}`)
	req := httptest.NewRequest("POST", "/patchtest", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	data, err := app.Storage.Get("patchtest")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// {"field2":"patched"} base64 encoded - only patches field2
	var patchStr = []byte(`{"data":"eyJmaWVsZDIiOiJwYXRjaGVkIn0="}`)
	req = httptest.NewRequest("PATCH", "/patchtest", bytes.NewBuffer(patchStr))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	patchedData, err := app.Storage.Get("patchtest")
	require.NoError(t, err)
	require.NotEqual(t, string(data), string(patchedData))
}

func TestRestPatchNotFound(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// {"field":"value"} base64 encoded
	var patchStr = []byte(`{"data":"eyJmaWVsZCI6InZhbHVlIn0="}`)
	req := httptest.NewRequest("PATCH", "/nonexistent", bytes.NewBuffer(patchStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRestPatchInvalidKey(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// {"field":"value"} base64 encoded
	var patchStr = []byte(`{"data":"eyJmaWVsZCI6InZhbHVlIn0="}`)
	req := httptest.NewRequest("PATCH", "/test/*", bytes.NewBuffer(patchStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRestPatchWriteFilterOnMergedData(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true

	// Write filter that requires "requiredField" to exist in the data
	// This would fail if filter was applied to partial patch data only
	app.WriteFilter("filteredpatch", func(key string, data []byte) ([]byte, error) {
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, err
		}
		if _, ok := obj["requiredField"]; !ok {
			return nil, errors.New("requiredField is missing")
		}
		return data, nil
	})

	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// Create initial data directly in storage (bypassing write filter for setup)
	// {"requiredField":"exists","otherField":"value1"} base64 encoded
	_, err := app.Storage.Set("filteredpatch", "eyJyZXF1aXJlZEZpZWxkIjoiZXhpc3RzIiwib3RoZXJGaWVsZCI6InZhbHVlMSJ9")
	require.NoError(t, err)

	// Patch with only otherField (no requiredField in patch): {"otherField":"patched"}
	// base64: eyJvdGhlckZpZWxkIjoicGF0Y2hlZCJ9
	// If filter was applied to patch data only, this would fail because requiredField is missing
	// But with merged data, requiredField exists from original data, so it should pass
	var patchStr = []byte(`{"data":"eyJvdGhlckZpZWxkIjoicGF0Y2hlZCJ9"}`)
	req := httptest.NewRequest("PATCH", "/filteredpatch", bytes.NewBuffer(patchStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the data was patched correctly
	patchedData, err := app.Storage.Get("filteredpatch")
	require.NoError(t, err)

	obj, err := objects.Decode(patchedData)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(obj.Data), &result)
	require.NoError(t, err)

	// Both fields should exist: requiredField from original, otherField patched
	require.Equal(t, "exists", result["requiredField"])
	require.Equal(t, "patched", result["otherField"])
}

func TestRestPatchPreservesCreated(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)

	// {"field1":"original"} base64 encoded
	index, err := app.Storage.Set("preservetest", "eyJmaWVsZDEiOiJvcmlnaW5hbCJ9")
	require.NoError(t, err)
	require.Equal(t, "preservetest", index)

	originalData, err := app.Storage.Get("preservetest")
	require.NoError(t, err)

	// {"field1":"patched"} base64 encoded
	var patchStr = []byte(`{"data":"eyJmaWVsZDEiOiJwYXRjaGVkIn0="}`)
	req := httptest.NewRequest("PATCH", "/preservetest", bytes.NewBuffer(patchStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	patchedData, err := app.Storage.Get("preservetest")
	require.NoError(t, err)

	require.NotEqual(t, string(originalData), string(patchedData))
}
