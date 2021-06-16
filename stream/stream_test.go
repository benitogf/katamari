package stream

import (
	"log"
	"net/http"
	"net/http/httptest"
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

func TestStreamObject(t *testing.T) {
	const testKey = "testing"
	const testData = `{"one": 2}`
	const testDataUpdated = `{"one": 1}`
	stream := Pools{
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

	wsConn, err := stream.New(testKey, testKey, w, req)
	require.NoError(t, err)
	require.Equal(t, 1, len(stream.Pools))
	require.Equal(t, testKey, stream.Pools[0].Key)
	require.Equal(t, 1, len(stream.Pools[0].connections))

	stream.SetCache(testKey, []byte(testData))

	cache, err := stream.GetCache(testKey)
	require.NoError(t, err)
	require.Equal(t, testData, string(cache.Data))
	require.NotZero(t, cache.Version)

	modifiedData, snapshot, version := stream.Patch(0, []byte(testDataUpdated))
	require.True(t, snapshot)
	require.NotZero(t, version)
	require.Equal(t, testDataUpdated, string(modifiedData))

	stream.Close(testKey, testKey, wsConn)
	require.Equal(t, 1, len(stream.Pools))
	require.Equal(t, testKey, stream.Pools[0].Key)
	require.Equal(t, 0, len(stream.Pools[0].connections))
}

func TestStreamList(t *testing.T) {
	const testKey = "testing/*"
	const testData = `[{"one": 11111111111111111},{"two": 222222222222222},{"three":3333333333333333333333}]`
	const testDataUpdated = `[{"one":11111111111111111},{"two":222222222222222},{"three":3333333333333333333333},{"four":4}]`
	const patchOperations = `[{"op":"add","path":"/3","value":{"four":4}}]`

	stream := Pools{
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

	wsConn, err := stream.New(testKey, testKey, w, req)
	require.NoError(t, err)
	require.Equal(t, 1, len(stream.Pools))
	require.Equal(t, testKey, stream.Pools[0].Key)
	require.Equal(t, 1, len(stream.Pools[0].connections))

	stream.SetCache(testKey, []byte(testData))

	cache, err := stream.GetCache(testKey)
	require.NoError(t, err)
	require.Equal(t, testData, string(cache.Data))
	require.NotZero(t, cache.Version)

	modifiedData, snapshot, version := stream.Patch(0, []byte(testDataUpdated))
	require.False(t, snapshot)
	require.NotZero(t, version)
	require.Equal(t, patchOperations, string(modifiedData))

	stream.Close(testKey, testKey, wsConn)
	require.Equal(t, 1, len(stream.Pools))
	require.Equal(t, testKey, stream.Pools[0].Key)
	require.Equal(t, 0, len(stream.Pools[0].connections))
}
