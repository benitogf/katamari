package samo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func Decode(message []byte) (string, error) {
	app := Server{}
	var wsEvent map[string]interface{}
	err := json.Unmarshal(message, &wsEvent)
	if err != nil {
		return "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(app.helpers.extractNonNil(wsEvent, "data"))
	if err != nil {
		return "", err
	}
	return strings.Trim(string(decoded), "\n"), nil
}

func TestRegex(t *testing.T) {
	app := Server{}
	separator := "/"
	rr, _ := regexp.Compile("^" + app.helpers.makeRouteRegex(separator) + "$")
	require.True(t, rr.MatchString("a/b/c"))
	require.False(t, rr.MatchString("/a/b/c"))
	require.False(t, rr.MatchString("a/b/c/"))
	require.False(t, rr.MatchString("a:b/c"))
	separator = ":"
	rr, _ = regexp.Compile("^" + app.helpers.makeRouteRegex(separator) + "$")
	require.True(t, rr.MatchString("a:b:c"))
	require.False(t, rr.MatchString("a:b/c"))
}

func TestValidKey(t *testing.T) {
	app := Server{}
	require.True(t, app.helpers.validKey("test", "/"))
	require.True(t, app.helpers.validKey("test/1", "/"))
	require.False(t, app.helpers.validKey("test//1", "/"))
	require.False(t, app.helpers.validKey("test///1", "/"))
}

func TestIsMo(t *testing.T) {
	app := Server{}
	require.True(t, app.helpers.isMO("thing", "thing/123", "/"))
	require.True(t, app.helpers.isMO("thing/123", "thing/123/123", "/"))
	require.False(t, app.helpers.isMO("thing/123", "thing/12", "/"))
	require.False(t, app.helpers.isMO("thing/1", "thing/123", "/"))
	require.False(t, app.helpers.isMO("thing/123", "thing/123/123/123", "/"))
}

func TestSASetGetDel(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	_ = app.storage.delData("sa", "test", "")
	index, err := app.storage.setData("sa", "test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ := app.storage.getData("sa", "test")
	var testObject Object
	err = json.Unmarshal(data, &testObject)
	require.NoError(t, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	err = app.storage.delData("sa", "test", "")
	require.NoError(t, err)
	raw, _ := app.storage.getData("sa", "test")
	dataDel := string(raw)
	require.Empty(t, dataDel)
}

func TestMOSetGetDel(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	_ = app.storage.delData("mo", "test", "MOtest")
	_ = app.storage.delData("mo", "test", "123")
	_ = app.storage.delData("mo", "test", "1")
	index, err := app.storage.setData("sa", "test/123", "123", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "123", index)
	index, err = app.storage.setData("mo", "test/MOtest", "MOtest", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "MOtest", index)
	_, _, index = app.helpers.makeIndexes("mo", "test", "", "T", app.separator)
	require.NotEmpty(t, index)
	data, _ := app.storage.getData("mo", "test")
	var testObjects []Object
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	require.Equal(t, "test", testObjects[0].Data)
}

func TestArchetype(t *testing.T) {
	app := Server{}
	app.Archetypes = Archetypes{
		"test1": func(data string) bool {
			return data == "test1"
		},
		"test?/*": func(data string) bool {
			return data == "test"
		},
	}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	require.False(t, app.helpers.checkArchetype("test1", "notest", app.Archetypes))
	require.True(t, app.helpers.checkArchetype("test1", "test1", app.Archetypes))
	require.False(t, app.helpers.checkArchetype("test1/1", "test1", app.Archetypes))
	require.False(t, app.helpers.checkArchetype("test0/1", "notest", app.Archetypes))
	require.True(t, app.helpers.checkArchetype("test0/1", "test", app.Archetypes))

	var jsonStr = []byte(`{"data":"notest"}`)
	req := httptest.NewRequest("POST", "/r/sa/test1", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestRPostNonObject(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`non object`)
	req := httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestRPostEmptyData(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`{"data":""}`)
	req := httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestAudit(t *testing.T) {
	app := Server{}
	app.Audit = func(r *http.Request) bool {
		return r.Header.Get("Upgrade") != "websocket" && r.Method != "GET" && r.Method != "DEL"
	}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)

	index, err := app.storage.setData("sa", "test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)

	req := httptest.NewRequest("GET", "/r/sa/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("DEL", "/r/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.console.err(err)
	require.Error(t, err)

	app.Audit = func(r *http.Request) bool {
		return r.Method == "GET"
	}

	var jsonStr = []byte(`{"data":"test"}`)
	req = httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 401, resp.StatusCode)

	req = httptest.NewRequest("GET", "/r/sa/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 200, resp.StatusCode)

}

func TestRPostKey(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", ':')
	defer app.Close(os.Interrupt)
	var jsonStr = []byte(`{"data":"test"}`)
	req := httptest.NewRequest("POST", "/r/sa/test::a", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestDoubleShutdown(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", ':')
	defer app.Close(os.Interrupt)
	app.Close(os.Interrupt)
}

func TestHttpRGet(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	_ = app.storage.delData("sa", "test", "")
	index, err := app.storage.setData("sa", "test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)
	data, _ := app.storage.getData("sa", "test")

	req := httptest.NewRequest("GET", "/r/sa/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, string(data), string(body))

	req = httptest.NewRequest("GET", "/r/sa/test/notest", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 404, resp.StatusCode)
}

func TestHttpRDel(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	_ = app.storage.delData("sa", "test", "")
	index, err := app.storage.setData("sa", "test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "test", index)

	req := httptest.NewRequest("DEL", "/r/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	data, _ := app.storage.getData("sa", "test")
	require.Equal(t, 204, resp.StatusCode)
	require.Empty(t, data)

	req = httptest.NewRequest("DEL", "/r/test", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 404, resp.StatusCode)
}

func TestRequestKey(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", ':')
	defer app.Close(os.Interrupt)
	require.NotEmpty(t, app.Server)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/:test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	app.console.err(err)
	require.Error(t, err)
	u2 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test::1"}
	c, _, err = websocket.DefaultDialer.Dial(u2.String(), nil)
	require.Nil(t, c)
	app.console.err(err)
	require.Error(t, err)
	u3 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test/1:"}
	c, _, err = websocket.DefaultDialer.Dial(u3.String(), nil)
	require.Nil(t, c)
	app.console.err(err)
	require.Error(t, err)

	req := httptest.NewRequest("GET", "/r/sa/test::1", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 400, resp.StatusCode)

	req = httptest.NewRequest("DEL", "/r/test::1", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	require.Equal(t, 400, resp.StatusCode)
}

func TestWsTime(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/time"}
	c1, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	c2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	count := 0
	c1time := ""
	c2time := ""

	go func() {
		for {
			_, message, err := c1.ReadMessage()
			if err != nil {
				app.console.err("read c1", err)
				break
			}
			c1time = string(message)
			app.console.log("time c1", c1time)
			count++
		}
	}()

	for {
		_, message, err := c2.ReadMessage()
		if err != nil {
			app.console.err("read c2", err)
			break
		}
		c2time = string(message)
		app.console.log("time c2", c2time)
		err = c2.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		require.NoError(t, err)
	}

	tryes := 0
	for count < 3 && tryes < 10000 {
		tryes++
		time.Sleep(2 * time.Millisecond)
	}

	err = c1.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)
	require.NotEmpty(t, c1time)
	require.NotEmpty(t, c2time)
}

func TestRPostWSBroadcast(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	_ = app.storage.delData("sa", "test", "")
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	started := false
	got := ""

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				app.console.err("read c", err)
				break
			}
			data, err := Decode(message)
			require.NoError(t, err)
			app.console.log("read c", data)
			if started {
				got = data
				err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				require.NoError(t, err)
			}
			started = true
		}
	}()

	tryes := 0
	for !started && tryes < 10000 {
		tryes++
		time.Sleep(2 * time.Millisecond)
	}
	var jsonStr = []byte(`{"data":"Buy coffee and bread for breakfast."}`)
	req := httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	tryes = 0
	for got == "" && tryes < 10000 {
		tryes++
		time.Sleep(2 * time.Millisecond)
	}
	var wsObject Object
	err = json.Unmarshal([]byte(got), &wsObject)
	require.NoError(t, err)
	var rPostObject Object
	err = json.Unmarshal(body, &rPostObject)
	require.NoError(t, err)
	require.Equal(t, wsObject.Index, rPostObject.Index)
	require.Equal(t, 200, resp.StatusCode)
}

func TestWSBroadcast(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	_ = app.storage.delData("mo", "test", "MOtest")
	_ = app.storage.delData("mo", "test", "123")
	_ = app.storage.delData("mo", "test", "1")
	index, err := app.storage.setData("sa", "test/1", "1", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "1", index)

	index, err = app.storage.setData("sa", "test/2", "2", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.Equal(t, "2", index)

	u1 := url.URL{Scheme: "ws", Host: app.address, Path: "/mo/test"}
	u2 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test/1"}
	c1, _, err := websocket.DefaultDialer.Dial(u1.String(), nil)
	require.NoError(t, err)
	c2, _, err := websocket.DefaultDialer.Dial(u2.String(), nil)
	require.NoError(t, err)
	wrote := false
	readCount := 0
	got1 := ""
	got2 := ""

	go func() {
		for {
			_, message, err := c1.ReadMessage()
			if err != nil {
				app.console.err("read c1", err)
				break
			}
			data, err := Decode(message)
			require.NoError(t, err)
			app.console.log("read c1", data)
			if readCount == 2 {
				got1 = data
				err = c1.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				require.NoError(t, err)
			}
			if readCount == 0 {
				err = c1.WriteMessage(websocket.TextMessage, []byte("{"+
					"\"op\": \"DEL\","+
					"\"index\": \"2\""+
					"}"))
				require.NoError(t, err)
			}
			readCount++
		}
	}()

	tryes := 0
	for readCount < 2 && tryes < 10000 {
		tryes++
		time.Sleep(2 * time.Millisecond)
	}

	for {
		_, message, err := c2.ReadMessage()
		if err != nil {
			app.console.err("read", err)
			break
		}
		data, err := Decode(message)
		require.NoError(t, err)
		app.console.log("read c2", data)
		if wrote {
			got2 = data
			err = c2.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			require.NoError(t, err)
		} else {
			app.console.log("writing from c2")
			err = c2.WriteMessage(websocket.TextMessage, []byte("{"+
				"\"index\": \"1\","+
				"\"data\": \"test2\""+
				"}"))
			require.NoError(t, err)
			wrote = true
		}
	}

	tryes = 0
	for got1 == "" && tryes < 10000 {
		tryes++
		time.Sleep(2 * time.Millisecond)
	}

	require.Equal(t, got1, "["+got2+"]")
}

func TestWSSADel(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)
	_ = app.storage.delData("sa", "test", "")
	index, err := app.storage.setData("sa", "test", "test", time.Now().UTC().UnixNano(), "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	started := false

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			app.console.err("read c", err)
			break
		}
		data, err := Decode(message)
		require.NoError(t, err)
		app.console.log("read c", data)
		if started {
			err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			require.NoError(t, err)
		} else {
			err = c.WriteMessage(websocket.TextMessage, []byte("{"+
				"\"op\": \"DEL\""+
				"}"))
			require.NoError(t, err)
			started = true
		}
	}

	req := httptest.NewRequest("GET", "/r/sa/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, 404, resp.StatusCode)
}

func TestBadSocketRequest(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)

	req := httptest.NewRequest("GET", "/sa/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, 400, resp.StatusCode)
}

func TestGetStats(t *testing.T) {
	app := Server{}
	app.Start("localhost:9889", "test/db", '/')
	defer app.Close(os.Interrupt)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"keys\":[\"test/1\"]}", string(body))

	_ = app.storage.delData("sa", "test/1", "")

	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "{\"keys\":[]}", string(body))
}
