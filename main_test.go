package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
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
	var wsEvent map[string]interface{}
	err := json.Unmarshal(message, &wsEvent)
	if err != nil {
		return "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(extractNonNil(wsEvent, "data"))
	if err != nil {
		return "", err
	}
	return strings.Trim(string(decoded), "\n"), nil
}

func TestRegex(t *testing.T) {
	separator := "/"
	rr, _ := regexp.Compile("^" + generateRouteRegex(separator) + "$")
	require.True(t, rr.MatchString("a/b/c"))
	require.False(t, rr.MatchString("/a/b/c"))
	require.False(t, rr.MatchString("a/b/c/"))
	require.False(t, rr.MatchString("a:b/c"))
	separator = ":"
	rr, _ = regexp.Compile("^" + generateRouteRegex(separator) + "$")
	require.True(t, rr.MatchString("a:b:c"))
	require.False(t, rr.MatchString("a:b/c"))
}

func TestValidKey(t *testing.T) {
	require.True(t, validKey("test", "/"))
	require.True(t, validKey("test/1", "/"))
	require.False(t, validKey("test//1", "/"))
	require.False(t, validKey("test///1", "/"))
}

func TestIsMo(t *testing.T) {
	require.True(t, isMO("thing", "thing/123", "/"))
	require.True(t, isMO("thing/123", "thing/123/123", "/"))
	require.False(t, isMO("thing/123", "thing/12", "/"))
	require.False(t, isMO("thing/1", "thing/123", "/"))
	require.False(t, isMO("thing/123", "thing/123/123/123", "/"))
}

func TestSASetGetDel(t *testing.T) {
	app := SAMO{}
	app.init("localhost:9889", "test/db", "/")
	go app.start()
	defer app.close(os.Interrupt)
	app.delData("sa", "test", "")
	index := app.setData("sa", "test", "", "", "test")
	require.NotEmpty(t, index)
	data := app.getData("sa", "test")
	var testObject Object
	err := json.Unmarshal(data, &testObject)
	require.NoError(t, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	app.delData("sa", "test", "")
	dataDel := string(app.getData("sa", "test"))
	require.Equal(t, "", dataDel)
}

func TestMOSetGetDel(t *testing.T) {
	app := SAMO{}
	app.init("localhost:9889", "test/db", "/")
	go app.start()
	defer app.close(os.Interrupt)
	app.delData("mo", "test", "MOtest")
	app.delData("mo", "test", "123")
	app.delData("mo", "test", "1")
	_ = app.setData("sa", "test/123", "", "", "test")
	index := app.setData("mo", "test", "MOtest", "", "test")
	require.Equal(t, "MOtest", index)
	data := app.getData("mo", "test")
	var testObjects []Object
	err := json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	require.Equal(t, "test", testObjects[0].Data)
}

func TestHttpRGet(t *testing.T) {
	app := SAMO{}
	app.init("localhost:9889", "test/db", "/")
	go app.start()
	defer app.close(os.Interrupt)
	app.delData("sa", "test", "")
	_ = app.setData("sa", "test", "", "", "test")
	data := app.getData("sa", "test")

	req := httptest.NewRequest("GET", "/r/sa/test", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, string(data), string(body))
}

func TestWSKey(t *testing.T) {
	app := SAMO{}
	app.init("localhost:9889", "test/db", ":")
	go app.start()
	defer app.close(os.Interrupt)
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/:test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, c)
	require.Error(t, err)
	u2 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test::1"}
	c, _, err = websocket.DefaultDialer.Dial(u2.String(), nil)
	require.Nil(t, c)
	require.Error(t, err)
	u3 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test/1:"}
	c, _, err = websocket.DefaultDialer.Dial(u3.String(), nil)
	require.Nil(t, c)
	require.Error(t, err)
}

func TestRPostWSBroadcast(t *testing.T) {
	app := SAMO{}
	app.init("localhost:9889", "test/db", "/")
	go app.start()
	defer app.close(os.Interrupt)
	app.delData("sa", "test", "")
	u := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	wrote := false
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
			if wrote {
				got = data
				_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			}
		}
	}()

	var jsonStr = []byte(`{"data":"Buy coffee and bread for breakfast."}`)
	req := httptest.NewRequest("POST", "/r/sa/test", bytes.NewBuffer(jsonStr))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	wrote = true
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	tryes := 0
	for got == "" && tryes < 10 {
		tryes++
		time.Sleep(800 * time.Millisecond)
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
	app := SAMO{}
	app.init("localhost:9889", "test/db", "/")
	go app.start()
	defer app.close(os.Interrupt)
	app.delData("mo", "test", "MOtest")
	app.delData("mo", "test", "123")
	app.delData("mo", "test", "1")
	_ = app.setData("sa", "test/1", "", "", "test")
	u1 := url.URL{Scheme: "ws", Host: app.address, Path: "/mo/test"}
	u2 := url.URL{Scheme: "ws", Host: app.address, Path: "/sa/test/1"}
	c1, _, err := websocket.DefaultDialer.Dial(u1.String(), nil)
	require.NoError(t, err)
	c2, _, err := websocket.DefaultDialer.Dial(u2.String(), nil)
	require.NoError(t, err)
	wrote := false
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
			if wrote {
				got1 = data
			}
		}
	}()

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
			_ = c2.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			_ = c1.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		} else {
			app.console.log("writing from c2")
			err = c2.WriteMessage(websocket.TextMessage, []byte("{"+
				"\"data\": \"test2\""+
				"}"))
			require.NoError(t, err)
			wrote = true
		}
	}

	require.Equal(t, got1, "["+got2+"]")
}
