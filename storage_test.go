package samo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func StorageSA(app *Server, t *testing.T) {
	app.Storage.Clear()
	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ := app.Storage.Get("sa", "test")
	testObject, err := app.objects.decode(data)
	require.NoError(t, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	index, err = app.Storage.Set("test", "test_update")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, err = app.Storage.Get("sa", "test")
	require.NoError(t, err)
	testObject, err = app.objects.decode(data)
	require.NoError(t, err)
	require.Equal(t, "test_update", testObject.Data)
	err = app.Storage.Del("test")
	require.NoError(t, err)
	raw, _ := app.Storage.Get("sa", "test")
	dataDel := string(raw)
	require.Empty(t, dataDel)
}

func StorageMO(app *Server, t *testing.T, testData string) {
	app.Storage.Clear()
	modData := testData + testData
	key, err := app.Storage.Set("test/123", testData)
	require.NoError(t, err)
	require.Equal(t, "123", key)
	key, err = app.Storage.Set("test/456", modData)
	require.NoError(t, err)
	require.Equal(t, "456", key)
	data, err := app.Storage.Get("mo", "test")
	require.NoError(t, err)
	var testObjects []Object
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	for i := range testObjects {
		if testObjects[i].Index == "123" {
			require.Equal(t, testData, testObjects[i].Data)
		}

		if testObjects[i].Index == "456" {
			require.Equal(t, modData, testObjects[i].Data)
		}
	}
	data1, err := app.Storage.Get("sa", "test/123")
	require.NoError(t, err)
	data2, err := app.Storage.Get("sa", "test/456")
	require.NoError(t, err)
	obj1, err := app.objects.decode(data1)
	require.NoError(t, err)
	obj2, err := app.objects.decode(data2)
	require.NoError(t, err)
	require.Equal(t, testData, obj1.Data)
	require.Equal(t, modData, obj2.Data)
	keys, err := app.Storage.Keys()
	require.NoError(t, err)
	require.Equal(t, "{\"keys\":[\"test/123\",\"test/456\"]}", string(keys))

	req := httptest.NewRequest(
		"POST", "/test/*",
		bytes.NewBuffer(
			[]byte(`{"data":"testpost"}`),
		),
	)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	dat, err := app.objects.decode(body)
	require.NoError(t, err)
	data, err = app.Storage.Get("mo", "test")
	app.console.Log(string(data))
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 3, len(testObjects))
	err = app.Storage.Del("test/" + dat.Index)
	require.NoError(t, err)
	data, err = app.Storage.Get("mo", "test")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	key, err = app.Storage.Set("test/glob1/glob123", testData)
	require.NoError(t, err)
	require.Equal(t, "glob123", key)
	key, err = app.Storage.Set("test/glob2/glob456", modData)
	require.NoError(t, err)
	require.Equal(t, "glob456", key)
	data, err = app.Storage.Get("mo", "test/*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	key, err = app.Storage.Set("test/1/glob/g123", testData)
	require.NoError(t, err)
	require.Equal(t, "g123", key)
	key, err = app.Storage.Set("test/2/glob/g456", modData)
	require.NoError(t, err)
	require.Equal(t, "g456", key)
	data, err = app.Storage.Get("mo", "test/*/glob/*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	key, err = app.Storage.Set("test1", testData)
	require.NoError(t, err)
	require.Equal(t, "test1", key)
	key, err = app.Storage.Set("test2", modData)
	require.NoError(t, err)
	require.Equal(t, "test2", key)
	data, err = app.Storage.Get("mo", "*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
}

var units = []string{
	"\xe4\xef\xf0\xe9\xf9l\x100",
	"V'\xe4\xc0\xbb>0\x86j",
	"0'\xe40\x860",
	"\b𝅗𝅝\x85",
	"𓏝",
	"𝅅",
	"'",
	"\xd80''",
	"\xd8%''",
	"0",
	"",
}

func TestStorageMemory(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageMO(app, t, app.messages.encode([]byte(units[i])))
	}
	StorageSA(app, t)
}

func TestStorageLeveldb(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Storage = &LevelStorage{}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageMO(app, t, app.messages.encode([]byte(units[i])))
	}
	StorageSA(app, t)
}
