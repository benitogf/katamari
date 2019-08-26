package etcd

import (
	"os"
	"testing"

	"github.com/benitogf/katamari"
)

func BenchmarkEtcdSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := katamari.Server{}
	app.Silence = true
	app.Storage = &Etcd{}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	katamari.StorageSetGetDelTest(app.Storage, b)
}
