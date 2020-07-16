package level

import (
	"os"
	"testing"

	katamari "bitbucket.org/idxgames/auth"
	"bitbucket.org/idxgames/auth/messages"
)

var units = []string{
	"\xe4\xef\xf0\xe9\xf9l\x100",
	"V'\xe4\xc0\xbb>0\x86j",
	"0'\xe40\x860",
	"\b𝅗𝅝\x85",
	"𓏝",
	"𝅅",
	"'",
	"\xd80''",
	"\xd8%''",
	"0",
	"",
}

func TestStorageLeveldb(t *testing.T) {
	t.Parallel()
	app := &katamari.Server{}
	app.Silence = true
	app.Storage = &Storage{Path: "test/db"}
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	for i := range units {
		katamari.StorageListTest(app, t, messages.Encode([]byte(units[i])))
	}
	katamari.StorageObjectTest(app, t)
}

func TestStreamBroadcastLevel(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Storage = &Storage{Path: "test/db1" + katamari.Time()}
	app.Start("localhost:0")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	katamari.StreamBroadcastTest(t, &app)
}

func TestStreamGlobBroadcastLevel(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Storage = &Storage{Path: "test/db2" + katamari.Time()}
	app.Start("localhost:0")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	katamari.StreamGlobBroadcastTest(t, &app)
}

func TestStreamBroadcastFilter(t *testing.T) {
	t.Parallel()
	app := katamari.Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Storage = &Storage{Path: "test/db3" + katamari.Time()}
	defer app.Close(os.Interrupt)
	katamari.StreamBroadcastFilterTest(t, &app)
}

func TestGetN(t *testing.T) {
	t.Parallel()
	app := &katamari.Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	katamari.StorageGetNTest(app, t)
}

func TestKeysRange(t *testing.T) {
	t.Parallel()
	app := &katamari.Server{}
	app.Silence = true
	app.Storage = &Storage{Path: "test/db4" + katamari.Time()}
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	katamari.StorageKeysRangeTest(app, t)
}
