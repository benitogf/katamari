package katamari

import (
	"os"
	"testing"

	"github.com/benitogf/katamari/messages"
)

func TestStorageMemory(t *testing.T) {
	t.Parallel()
	app := &Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageListTest(app, t, messages.Encode([]byte(units[i])))
	}
	StorageObjectTest(app, t)
}

func TestStreamBroadcastMemory(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	StreamBroadcastTest(t, &app)
}

func TestStreamGlobBroadcastMemory(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	StreamGlobBroadcastTest(t, &app)
}

func TestStreamBroadcastFilter(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	defer app.Close(os.Interrupt)
	StreamBroadcastFilterTest(t, &app)
}

func TestGetN(t *testing.T) {
	t.Parallel()
	app := &Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	StorageGetNTest(app, t)
}

func TestKeysRange(t *testing.T) {
	t.Parallel()
	app := &Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	StorageKeysRangeTest(app, t)
}

func TestStreamItemGlobBroadcastLevel(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Start("localhost:0")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	StreamItemGlobBroadcastTest(t, &app)
}
