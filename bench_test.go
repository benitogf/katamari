package samo

import (
	"bytes"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

func BenchmarkLevelDbStorageSetGet(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	os.RemoveAll("test")
	app.silence = true
	app.Storage = &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{Active: false}}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := strconv.FormatInt(int64(i), 10)
		_, _ = app.Storage.Set("test/"+index, index, 0, "test"+index)
		_, _ = app.Storage.Get("sa", "test/"+index)
	}
}

func BenchmarkLevelDbStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	os.RemoveAll("test")
	app.silence = true
	app.Storage = &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{Active: false}}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(
			"POST", "/r/mo/test",
			bytes.NewBuffer(
				[]byte(`{"data":"test`+strconv.FormatInt(int64(i), 10)+`"}`),
			),
		)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
	}
}

func BenchmarkMemoryStorageSetGet(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := strconv.FormatInt(int64(i), 10)
		_, _ = app.Storage.Set("test/"+index, index, 0, "test"+index)
		_, _ = app.Storage.Get("sa", "test/"+index)
	}
}

func BenchmarkMemoryStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(
			"POST", "/r/mo/test",
			bytes.NewBuffer(
				[]byte(`{"data":test`+strconv.FormatInt(int64(i), 10)+`"}`),
			),
		)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
	}
}
