package katamari

import (
	"os"
	"testing"
)

func TestStorageMemory(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageListTest(app, t, app.messages.Encode([]byte(units[i])))
	}
	StorageObjectTest(app, t)
}

func TestStreamBroadcastMemory(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.NamedSocket = "ipctest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	StreamBroadcastTest(t, &app)
}

func TestStreamGlobBroadcastMemory(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.NamedSocket = "ipctest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	StreamGlobBroadcastTest(t, &app)
}
