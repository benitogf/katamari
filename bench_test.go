package samo

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func storagePost(ServeHTTP func(w http.ResponseWriter, req *http.Request), b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(
			"POST", "/r/mo/test",
			bytes.NewBuffer(
				[]byte(`{"data":"test`+(&Messages{}).encode([]byte(strconv.FormatInt(int64(i), 10)))+`"}`),
			),
		)
		w := httptest.NewRecorder()
		ServeHTTP(w, req)
		resp := w.Result()
		require.Equal(b, 200, resp.StatusCode)
	}
}

func BenchmarkMemorydbStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storagePost(app.Router.ServeHTTP, b)
}

func BenchmarkLevelStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &LevelStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storagePost(app.Router.ServeHTTP, b)
}

func BenchmarkEtcdStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Storage = &EtcdStorage{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storagePost(app.Router.ServeHTTP, b)
}

func storageSetGetDel(db Database, b *testing.B) {
	tests := make(map[string]string)
	for i := 0; i < b.N; i++ {
		index := strconv.FormatInt(int64(i), 10)
		tests["test"+index] = "test" + index
	}
	b.ResetTimer()
	for index := range tests {
		ci, _ := db.Set("test/"+index, index, 0, tests[index])
		_, _ = db.Get("sa", "test/"+ci)
		_ = db.Del("test/" + ci)
	}
	result, err := db.Get("mo", "test")
	require.NoError(b, err)
	require.Equal(b, "[]", string(result))
}

func BenchmarkMemoryStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b)
}

func BenchmarkLevelStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &LevelStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b)
}

func BenchmarkEtcdStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &EtcdStorage{}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b)
}
