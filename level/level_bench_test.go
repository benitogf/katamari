package level

import (
	"os"
	"testing"

	"github.com/benitogf/katamari"
)

func BenchmarkLevelStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := katamari.Server{}
	app.Silence = true
	app.Storage = &Storage{
		Path: "test/db"}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	katamari.StorageSetGetDelTest(app.Storage, b)
}
