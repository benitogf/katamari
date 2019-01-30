package samo

import (
	"bytes"
	"io/ioutil"
	"log"
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

func BenchmarkLevelDbStorageSetGet(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	os.RemoveAll("test")
	app.Silence = true
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
	app.Silence = true
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
		app.router.ServeHTTP(w, req)
	}
}

func BenchmarkMemoryStorageSetGet(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
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
	app.Silence = true
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
		app.router.ServeHTTP(w, req)
	}
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
						data, err := (&Helpers{}).Decode(message)
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
		app.router.ServeHTTP(w, req)
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

func Benchmark300Msgs10ClientBroadcast(b *testing.B) {
	multipleClientBroadcast(300, 10, 3000, b)
}

func Benchmark100Msgs100ClientBroadcast(b *testing.B) {
	multipleClientBroadcast(100, 100, 3000, b)
}

func Benchmark30Msgs1000ClientBroadcast(b *testing.B) {
	multipleClientBroadcast(30, 1000, 3000, b)
}
