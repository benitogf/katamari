package samo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func StorageSA(app *Server, t *testing.T, driver string) {
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test", 1, "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ := app.Storage.Get("sa", "test")
	testObject, err := app.objects.decode(data)
	require.NoError(t, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	// mariadb handles created/updated so it can't be mocked
	if driver != "mariadb" {
		require.Equal(t, int64(1), testObject.Created)
	}
	index, err = app.Storage.Set("test", "test", 2, "test_update")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ = app.Storage.Get("sa", "test")
	testObject, err = app.objects.decode(data)
	require.NoError(t, err)
	require.Equal(t, "test_update", testObject.Data)
	// mariadb handles created/updated so it can't be mocked
	if driver != "mariadb" {
		require.Equal(t, int64(2), testObject.Updated)
		require.Equal(t, int64(1), testObject.Created)
	}
	err = app.Storage.Del("test")
	require.NoError(t, err)
	raw, _ := app.Storage.Get("sa", "test")
	dataDel := string(raw)
	require.Empty(t, dataDel)
}

func StorageMO(app *Server, t *testing.T, testData string) {
	_ = app.Storage.Del("test/456")
	_ = app.Storage.Del("test/123")
	_ = app.Storage.Del("test/1")
	modData := testData + testData
	index, err := app.Storage.Set("test/123", "123", 0, testData)
	require.NoError(t, err)
	require.Equal(t, "123", index)
	index, err = app.Storage.Set("test/456", "456", 0, modData)
	require.NoError(t, err)
	require.Equal(t, "456", index)
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
		"POST", "/r/mo/test",
		bytes.NewBuffer(
			[]byte(`{"data":"testauto"}`),
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
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 3, len(testObjects))
	err = app.Storage.Del("test/" + dat.Index)
	require.NoError(t, err)
}

var units = []string{
	"\xe4\xef\xf0\xe9\xf9l\x100",
	"elIjoiYSB0aGluZyBpbi" +
		"B0aGUgYm94OiAxNTgxNm" +
		"YxYz\x12NMwOTA5NzU4MjZk" +
		"YzYwOWUxYzY5ZGI1Nzdh" +
		"YjViZmVjNTdjN2IwNTci" +
		"fQ==",
	"eyJuYW10=;O\xb6\x8d\xae\"\xeaaGlu" +
		"ZyBpbiBform-dataOiAx" +
		"NTgx\\mYx\x00\x04MwOTA5NzU4" +
		"MjZkQzYwOWUxYzY5ZGI1" +
		"NzdhYjViZmVjNTdjN2Tc" +
		"ifQ==",
	"eyYW10=;O\xb6\x8d\xae\"\xeaaGluZy" +
		"BpbiBform-dataOiAxNT" +
		"gx\\mYx\x00\x04MwOTA5NzU4Mj" +
		"ZkQzYwOWUxYzY5ZGI1Nz" +
		"dhYjViZmVjNTdjN2Tcif" +
		"Q==",
	"eW1lIjoiYSB0aGluZyBp" +
		"biB0aGU\x90Ym94OiAxNTgx" +
		"NmYxYzMwOTA5NzU4MjZk" +
		"YzYwOWUxYzY5ZGI1Nzdh" +
		"YjViZmVjNTdjN2IwNTci" +
		"fQ==",
	`"eW1lIjoiYSB0aGluZyBp""eW1lIjoiYSB0aGluZyBp"`,
	`"""eW1lIjoiYSB0aGluZyBp"""`,
	`"`,
	`""`,
	`"""`,
	`eW1lIjoiYSB0aGluZyBp"`,
	`"eW1lIjoiYSB0aGluZyBp"`,
	`eW1lIjoiYSB0aGluZyBp"`,
	`eW1lIjoiYSB0aGluZyBp"eW1lIjoiYSB0aGluZyBp`,
	``,
	"=yifQe\x05",
}

func TestStorageMemory(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageMO(app, t, fmt.Sprintf("%#v", units[i]))
	}
	StorageSA(app, t, "memory")
}

func TestStorageLeveldb(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Storage = &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{Active: false}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageMO(app, t, fmt.Sprintf("%#v", units[i]))
	}
	StorageSA(app, t, "leveldb")
}

func TestStorageMariadb(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Storage = &MariaDbStorage{
		User:     "root",
		Password: "",
		Name:     "samo",
		Storage:  &Storage{Active: false}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageMO(app, t, fmt.Sprintf("%#v", units[i]))
		time.Sleep(200 * time.Millisecond)
	}
	StorageSA(app, t, "mariadb")
}
