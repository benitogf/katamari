// +build etcd

package samo

import (
	"os"
	"testing"
)

func TestStorageEtcd(t *testing.T) {
	app := &Server{}
	app.Silence = true
	app.Storage = &EtcdStorage{}
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	for i := range units {
		StorageMO(app, t, app.messages.encode([]byte(units[i])))
	}
	StorageSA(app, t)
}

func TestWsRestBroadcastEtcd(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Storage = &EtcdStorage{}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	wsRestBroadcast(t, &app)
}

func TestWsBroadcastEtcd(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Storage = &EtcdStorage{}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	wsBroadcast(t, &app)
}
