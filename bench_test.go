package samo

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func storagePost(ServeHTTP func(w http.ResponseWriter, req *http.Request), b *testing.B, storage string) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(
			"POST", "/r/mo/test",
			bytes.NewBuffer(
				[]byte(`{"data":"test`+strconv.FormatInt(int64(i), 10)+`"}`),
			),
		)
		w := httptest.NewRecorder()
		ServeHTTP(w, req)
		resp := w.Result()
		require.Equal(b, 200, resp.StatusCode)
		if storage == "mariadb" {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func BenchmarkMemorydbStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storagePost(app.Router.ServeHTTP, b, "memory")
}

func BenchmarkLeveldbStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{Active: false}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storagePost(app.Router.ServeHTTP, b, "leveldb")
}
func BenchmarkMariadbStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &MariaDbStorage{
		User:     "root",
		Password: "",
		Name:     "samo",
		Storage:  &Storage{Active: false}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storagePost(app.Router.ServeHTTP, b, "mariadb")
}

func storageSetGetDel(db Database, b *testing.B, storage string) {
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
		// https://stackoverflow.com/questions/14331032/mysql-error-1040-too-many-connection/34176072
		if storage == "mariadb" {
			time.Sleep(100 * time.Millisecond)
		}
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
	storageSetGetDel(app.Storage, b, "memory")
}

func BenchmarkLevelDbStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{Active: false}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b, "leveldb")
}

func BenchmarkMariadbStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &MariaDbStorage{
		User:     "root",
		Password: "",
		Name:     "samo",
		Storage:  &Storage{Active: false}}
	app.Storage.Clear()
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b, "mariadb")
}

func multipleClientBroadcast(numberOfMsgs int, numberOfClients int, timeout int, b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	u1 := url.URL{Scheme: "ws", Host: app.address, Path: "/mo/test"}
	var ops int64
	ops = 0
	b.ResetTimer()
	for i := 0; i < numberOfClients; i++ {
		go func(i int) {
			conn, _, err := websocket.DefaultDialer.Dial(u1.String(), nil)
			if err != nil {
				log.Fatal(err, atomic.LoadInt64(&ops))
			}
			go func(conn *websocket.Conn) {
				count := 0
				for {
					_, message, err := conn.ReadMessage()
					if err != nil {
						app.console.Err("read c"+strconv.Itoa(i), err)
						break
					}
					go func(message []byte) {
						data, err := app.messages.read(message)
						if err != nil {
							log.Fatal(err)
						}
						atomic.AddInt64(&ops, 1)
						app.console.Log("read c"+strconv.Itoa(i), data)
					}(message)
					count++
					if count == numberOfMsgs {
						break
					}
				}
				conn.Close()
			}(conn)
		}(i)
	}

	tryes := 0
	for atomic.LoadInt64(&ops) < int64(numberOfClients) && tryes < timeout {
		tryes++
		time.Sleep(1 * time.Millisecond)
	}

	var jsonStr = []byte(`{"data":"test..."}`)
	for i := 2; i <= numberOfMsgs; i++ {
		req := httptest.NewRequest("POST", "/r/mo/test", bytes.NewBuffer(jsonStr))
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		resp := w.Result()
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		tryes = 0
		for atomic.LoadInt64(&ops) < int64(numberOfClients*i) && tryes < timeout {
			tryes++
			time.Sleep(1 * time.Millisecond)
		}
	}

	require.Equal(b, int64(numberOfClients*numberOfMsgs), atomic.LoadInt64(&ops))
}

func Benchmark100Msgs10ClientBroadcast(b *testing.B) {
	multipleClientBroadcast(100, 10, 3000, b)
}

func Benchmark100Msgs100ClientBroadcast(b *testing.B) {
	multipleClientBroadcast(100, 100, 3000, b)
}
