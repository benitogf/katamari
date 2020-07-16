package katamari

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
	"github.com/stretchr/testify/require"
)

// StorageObjectTest testing storage function
func StorageObjectTest(app *Server, t *testing.T) {
	app.Storage.Clear()
	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ := app.Storage.Get("test")
	testObject, err := objects.Decode(data)
	require.NoError(t, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	index, err = app.Storage.Set("test", "test_update")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, err = app.Storage.Get("test")
	require.NoError(t, err)
	testObject, err = objects.Decode(data)
	require.NoError(t, err)
	require.Equal(t, "test_update", testObject.Data)
	err = app.Storage.Del("test")
	require.NoError(t, err)
	raw, _ := app.Storage.Get("test")
	dataDel := string(raw)
	require.Empty(t, dataDel)
}

// StorageListTest testing storage function
func StorageListTest(app *Server, t *testing.T, testData string) {
	app.Storage.Clear()
	modData := testData + testData
	key, err := app.Storage.Set("test/123", testData)
	require.NoError(t, err)
	require.Equal(t, "123", key)
	key, err = app.Storage.Set("test/456", modData)
	require.NoError(t, err)
	require.Equal(t, "456", key)
	data, err := app.Storage.Get("test/*")
	require.NoError(t, err)
	var testObjects []objects.Object
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
	data1, err := app.Storage.Get("test/123")
	require.NoError(t, err)
	data2, err := app.Storage.Get("test/456")
	require.NoError(t, err)
	obj1, err := objects.Decode(data1)
	require.NoError(t, err)
	obj2, err := objects.Decode(data2)
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
	dat, err := objects.Decode(body)
	require.NoError(t, err)
	data, err = app.Storage.Get("test/*")
	app.console.Log(string(data))
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 3, len(testObjects))
	err = app.Storage.Del("test/" + dat.Index)
	require.NoError(t, err)
	data, err = app.Storage.Get("test/*")
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
	data, err = app.Storage.Get("test/*/*")
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
	data, err = app.Storage.Get("test/*/glob/*")
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
	data, err = app.Storage.Get("*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	err = app.Storage.Del("*")
	require.NoError(t, err)
	data, err = app.Storage.Get("*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 0, len(testObjects))
}

// StorageSetGetDelTest testing storage function
func StorageSetGetDelTest(db Database, b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ci, err := db.Set("test/1", "test1")
		require.NoError(b, err)
		_, err = db.Get("test/" + ci)
		require.NoError(b, err)
		err = db.Del("test/" + ci)
		result, err := db.Get("test/*")
		require.NoError(b, err)
		require.Equal(b, "[]", string(result))
	}
}

// StorageGetNTest testing storage GetN function
func StorageGetNTest(app *Server, t *testing.T) {
	app.Storage.Clear()
	testData := base64.StdEncoding.EncodeToString([]byte(units[0]))
	for i := 0; i < 100; i++ {
		value := strconv.Itoa(i)
		key, err := app.Storage.Set("test/"+value, testData)
		require.NoError(t, err)
		require.Equal(t, value, key)
		time.Sleep(time.Millisecond * 1)
	}

	limit := 1
	testObjects, err := app.Storage.GetN("test/*", limit)
	require.NoError(t, err)
	require.Equal(t, limit, len(testObjects))
	require.Equal(t, "99", testObjects[0].Index)
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

// StorageKeysRangeTest testing storage GetN function
func StorageKeysRangeTest(app *Server, t *testing.T) {
	app.Storage.Clear()
	testData := units[0]
	first := ""
	for i := 0; i < 100; i++ {
		path := key.Build("test/*")
		key, err := app.Storage.Set(path, testData)
		if first == "" {
			first = key
		}
		require.NoError(t, err)
		require.Equal(t, path, "test/"+key)
		time.Sleep(time.Nanosecond * 1)
	}

	keys, err := app.Storage.KeysRange("test/*", 0, key.Decode(first))
	require.NoError(t, err)
	require.Equal(t, 1, len(keys))
	require.Equal(t, "test/"+first, keys[0])
}
