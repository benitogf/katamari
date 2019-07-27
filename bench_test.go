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

func BenchmarkRedisStoragePost(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &RedisStorage{
		Storage: &Storage{}}
	app.Start("localhost:9889")
	app.Storage.Clear()
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

func BenchmarkRedisStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &RedisStorage{
		Storage: &Storage{}}
	app.Start("localhost:9889")
	app.Storage.Clear()
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
						event, err := app.messages.decode(message)
						if err != nil {
							log.Fatal(err)
						}
						atomic.AddInt64(&ops, 1)
						app.console.Log("read c"+strconv.Itoa(i), event.Data)
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

	var jsonStr = []byte(`{"data":"` + app.messages.encode([]byte("test...")) + `"}`)
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

func Benchmark10Msgs10ClientBroadcast(b *testing.B) {
	multipleClientBroadcast(10, 10, 3000, b)
}
