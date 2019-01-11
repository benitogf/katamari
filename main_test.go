package main

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidKey(t *testing.T) {
	require.Equal(t, false, validKey("/invalid//invalid", "/"))
	require.Equal(t, false, validKey("invalid//invalid", "/"))
	require.Equal(t, false, validKey("invalid/", "/"))
	require.Equal(t, false, validKey("/invalid", "/"))
	require.Equal(t, false, validKey("/invalid/", "/"))
	require.Equal(t, false, validKey("//invalid/", "/"))
	require.Equal(t, false, validKey("invalid///invalid", "/"))
	require.Equal(t, true, validKey("valid", "/"))
	require.Equal(t, true, validKey("valid/valid", "/"))
}

func TestIsMo(t *testing.T) {
	require.Equal(t, true, isMO("thing", "thing/123", "/"))
	require.Equal(t, true, isMO("thing/123", "thing/123/123", "/"))
	require.Equal(t, false, isMO("thing/123", "thing/12", "/"))
	require.Equal(t, false, isMO("thing/1", "thing/123", "/"))
	require.Equal(t, false, isMO("thing/123", "thing/123/123/123", "/"))
}

func TestSASetGetDel(t *testing.T) {
	app := SAMO{}
	app.init("localhost:9889", "test/db", "/")
	go app.start()
	app.delData("sa", "test", "")
	index := app.setData("sa", "test", "", "", "test")
	require.NotEqual(t, "", index)
	data := app.getData("sa", "test")
	var testObject Object
	err := json.Unmarshal(data, &testObject)
	require.Equal(t, nil, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	app.delData("sa", "test", "")
	dataDel := string(app.getData("sa", "test"))
	require.Equal(t, "", dataDel)
	app.close(os.Interrupt)
}

func TestMOSetGetDel(t *testing.T) {
	app := SAMO{}
	app.init("localhost:9889", "test/db", "/")
	go app.start()
	app.delData("mo", "test", "MOtest")
	app.delData("mo", "test", "123")
	_ = app.setData("sa", "test/123", "", "", "test")
	index := app.setData("mo", "test", "MOtest", "", "test")
	require.Equal(t, "MOtest", index)
	data := app.getData("mo", "test")
	var testObjects []Object
	err := json.Unmarshal(data, &testObjects)
	require.Equal(t, nil, err)
	require.Equal(t, 2, len(testObjects))
	require.Equal(t, "test", testObjects[0].Data)
	app.close(os.Interrupt)
}

func TestRegex(t *testing.T) {
	separator := "/"
	rr, _ := regexp.Compile("^" + generateRouteRegex(separator) + "$")
	require.Equal(t, true, rr.MatchString("a/b/c"))
	require.Equal(t, false, rr.MatchString("a:b/c"))
	separator = ":"
	rr, _ = regexp.Compile("^" + generateRouteRegex(separator) + "$")
	require.Equal(t, true, rr.MatchString("a:b:c"))
	require.Equal(t, false, rr.MatchString("a:b/c"))
}
