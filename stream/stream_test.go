package stream

import (
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/benitogf/coat"
	hjhttptest "github.com/getlantern/httptest"
	"github.com/stretchr/testify/require"
)

func makeStreamRequestMock(url string) (*http.Request, *hjhttptest.HijackableResponseRecorder) {
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Add("Connection", "upgrade")
	req.Header.Add("Sec-Websocket-Version", "13")
	req.Header.Add("Sec-Websocket-Key", "alo")
	req.Header.Add("Upgrade", "websocket")
	w := hjhttptest.NewRecorder(nil)

	return req, w
}

const domain = "http://example.com"

func TestSnapshot(t *testing.T) {
	const testKey = "testing"
	const testData = `{"one": 2}`
	const testDataUpdated = `{"one": 1}`
	stream := Stream{
		Console: coat.NewConsole(domain, false),
		OnSubscribe: func(key string) error {
			log.Println("sub", key)
			return nil
		},
		OnUnsubscribe: func(key string) {
			log.Println("unsub", key)
		},
	}

	req, w := makeStreamRequestMock(domain + "/" + testKey)

	wsConn, err := stream.New(testKey, w, req)
	require.NoError(t, err)
	require.Equal(t, 1, len(stream.pools))
	require.Equal(t, testKey, stream.pools[0].Key)
	require.Equal(t, 1, len(stream.pools[0].connections))

	stream.setCache(testKey, []byte(testData))

	cacheVersion, err := stream.GetCacheVersion(testKey)
	require.NoError(t, err)
	require.NotZero(t, cacheVersion)

	modifiedData, snapshot, version := stream.Patch(0, []byte(testDataUpdated))
	require.True(t, snapshot)
	require.NotZero(t, version)
	require.Equal(t, testDataUpdated, string(modifiedData))

	stream.Close(testKey, wsConn)
	require.Equal(t, 1, len(stream.pools))
	require.Equal(t, testKey, stream.pools[0].Key)
	require.Equal(t, 0, len(stream.pools[0].connections))
}

func TestPatch(t *testing.T) {
	const testKey = "testing/*"
	const testData = `[{"one": 11111111111111111},{"two": 222222222222222},{"three":3333333333333333333333}]`
	const testDataUpdated = `[{"one":11111111111111111},{"two":222222222222222},{"three":3333333333333333333333},{"four":4}]`
	const patchOperations = `[{"op":"add","path":"/3","value":{"four":4}}]`

	stream := Stream{
		Console: coat.NewConsole(domain, false),
		OnSubscribe: func(key string) error {
			log.Println("sub", key)
			return nil
		},
		OnUnsubscribe: func(key string) {
			log.Println("unsub", key)
		},
	}

	req, w := makeStreamRequestMock(domain + "/" + testKey)

	wsConn, err := stream.New(testKey, w, req)
	require.NoError(t, err)
	require.Equal(t, 1, len(stream.pools))
	require.Equal(t, testKey, stream.pools[0].Key)
	require.Equal(t, 1, len(stream.pools[0].connections))

	stream.setCache(testKey, []byte(testData))

	cacheVersion, err := stream.GetCacheVersion(testKey)
	require.NoError(t, err)
	require.NotZero(t, cacheVersion)

	modifiedData, snapshot, version := stream.Patch(0, []byte(testDataUpdated))
	require.False(t, snapshot)
	require.NotZero(t, version)
	require.Equal(t, patchOperations, string(modifiedData))

	stream.Close(testKey, wsConn)
	require.Equal(t, 1, len(stream.pools))
	require.Equal(t, testKey, stream.pools[0].Key)
	require.Equal(t, 0, len(stream.pools[0].connections))
}

func TestConcurrentBroadcast(t *testing.T) {
	const testData = `[{"one": 11111111111111111},{"two": 222222222222222},{"three":3333333333333333333333}]`
	var wg sync.WaitGroup

	stream := Stream{
		Console: coat.NewConsole(domain, false),
		OnSubscribe: func(key string) error {
			log.Println("sub", key)
			return nil
		},
		OnUnsubscribe: func(key string) {
			log.Println("unsub", key)
		},
	}

	req, w := makeStreamRequestMock(domain + "/")
	wsConn, err := stream.New("", w, req)
	require.NoError(t, err)
	require.Equal(t, 1, len(stream.pools))
	require.Equal(t, "", stream.pools[0].Key)
	require.Equal(t, 1, len(stream.pools[0].connections))

	reqA, wA := makeStreamRequestMock(domain + "/a")
	wsConnA, err := stream.New("a", wA, reqA)
	require.NoError(t, err)
	require.Equal(t, 2, len(stream.pools))
	require.Equal(t, "a", stream.pools[1].Key)
	require.Equal(t, 1, len(stream.pools[1].connections))

	reqB, wB := makeStreamRequestMock(domain + "/b")
	wsConnB, err := stream.New("b", wB, reqB)
	require.NoError(t, err)
	require.Equal(t, 3, len(stream.pools))
	require.Equal(t, "b", stream.pools[2].Key)
	require.Equal(t, 1, len(stream.pools[2].connections))

	stream.setCache("a", []byte(testData))
	stream.setCache("b", []byte(testData))

	fakeGet := func(key string) ([]byte, error) {
		return []byte(testData), nil
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			stream.Broadcast("a", BroadcastOpt{
				Get: fakeGet,
				Callback: func() {
					wg.Done()
				},
			})
		}()
	}

	for y := 0; y < 10; y++ {
		wg.Add(1)
		go func() {
			go stream.Broadcast("b", BroadcastOpt{
				Get: fakeGet,
				Callback: func() {
					wg.Done()
				},
			})
		}()
	}

	wg.Wait()

	stream.Close("", wsConn)
	stream.Close("a", wsConnA)
	stream.Close("b", wsConnB)
	require.Equal(t, 3, len(stream.pools))
	require.Equal(t, 0, len(stream.pools[0].connections))
	require.Equal(t, 0, len(stream.pools[1].connections))
}
