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
	app.NamedSocket = "ipctest1" + app.Time()
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	StreamBroadcastTest(t, &app)
}

func TestStreamGlobBroadcastMemory(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.NamedSocket = "ipctest2" + app.Time()
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	StreamGlobBroadcastTest(t, &app)
}

func TestStreamBroadcastFilter(t *testing.T) {
	t.Parallel()
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.NamedSocket = "ipctest" + app.Time()
	defer app.Close(os.Interrupt)
	StreamBroadcastFilterTest(t, &app)
}
