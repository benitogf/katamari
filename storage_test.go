package samo

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func StorageSA(app *Server, t *testing.T) {
	_ = app.Storage.Del("test")
	index, err := app.Storage.Set("test", "test", 0, "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ := app.Storage.Get("sa", "test")
	var testObject Object
	err = json.Unmarshal(data, &testObject)
	require.NoError(t, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	err = app.Storage.Del("test")
	require.NoError(t, err)
	raw, _ := app.Storage.Get("sa", "test")
	dataDel := string(raw)
	require.Empty(t, dataDel)
}

func StorageMO(app *Server, t *testing.T) {
	_ = app.Storage.Del("test/MOtest")
	_ = app.Storage.Del("test/123")
	_ = app.Storage.Del("test/1")
	index, err := app.Storage.Set("test/123", "123", 0, "test")
	require.NoError(t, err)
	require.Equal(t, "123", index)
	index, err = app.Storage.Set("test/MOtest", "MOtest", 0, "test")
	require.NoError(t, err)
	require.Equal(t, "MOtest", index)
	data, err := app.Storage.Get("mo", "test")
	require.NoError(t, err)
	var testObjects []Object
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	require.Equal(t, "test", testObjects[0].Data)
	keys, err := app.Storage.Keys()
	require.NoError(t, err)
	require.Equal(t, "{\"keys\":[\"test/123\",\"test/MOtest\"]}", string(keys))

}

func TestStorageMemory(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	StorageMO(app, t)
	StorageSA(app, t)
}

func TestStorageLeveldb(t *testing.T) {
	os.RemoveAll("test")
	app := &Server{}
	app.Silence = true
	app.Storage = &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{Active: false}}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	StorageMO(app, t)
	StorageSA(app, t)
}
