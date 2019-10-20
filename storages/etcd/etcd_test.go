package etcd

import (
	"os"
	"testing"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/messages"
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

func TestStorageEtcd(t *testing.T) {
	app := katamari.Server{}
	app.Silence = true
	app.Storage = &Etcd{}
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	for i := range units {
		katamari.StorageListTest(&app, t, messages.Encode([]byte(units[i])))
	}
	katamari.StorageObjectTest(&app, t)
}

func TestStreamBroadcastEtcd(t *testing.T) {
	app := katamari.Server{}
	app.Silence = true
	app.ForcePatch = true
	app.NamedSocket = "ipctest"
	app.Storage = &Etcd{}
	app.Start("localhost:0")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	katamari.StreamBroadcastTest(t, &app)
}

func TestStreamGlobBroadcastEtcd(t *testing.T) {
	app := katamari.Server{}
	app.Silence = true
	app.ForcePatch = true
	app.NamedSocket = "ipctest"
	app.Storage = &Etcd{}
	app.Start("localhost:0")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	katamari.StreamGlobBroadcastTest(t, &app)
}
