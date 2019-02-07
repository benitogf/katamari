package samo

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func StorageSA(app *Server, t *testing.T, driver string) {
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test", 1, "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ := app.Storage.Get("sa", "test")
	testObject, err := app.objects.read(data)
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
	testObject, err = app.objects.read(data)
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

func StorageMO(app *Server, t *testing.T) {
	_ = app.Storage.Del("test/456")
	_ = app.Storage.Del("test/123")
	_ = app.Storage.Del("test/1")
	testData := `	"\xe4\xef\xf0\xe9\xf9l\x100"
`
	index, err := app.Storage.Set("test/123", "123", 0, testData)
	require.NoError(t, err)
	require.Equal(t, "123", index)
	index, err = app.Storage.Set("test/456", "456", 0, testData)
	require.NoError(t, err)
	require.Equal(t, "456", index)
	data, err := app.Storage.Get("mo", "test")
	require.NoError(t, err)
	var testObjects []Object
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	require.Equal(t, testData, testObjects[0].Data)
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
	data, err = app.Storage.Get("mo", "test")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 3, len(testObjects))
}

func TestStorageMemory(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	StorageMO(app, t)
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
	StorageMO(app, t)
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
	StorageMO(app, t)
	StorageSA(app, t, "mariadb")
}
