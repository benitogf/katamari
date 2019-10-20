package katamari

import (
	"os"
	"testing"
)

// go test -bench -run=^$

func BenchmarkMemoryStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	var app = Server{}
	app.Silence = true
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	StorageSetGetDelTest(app.Storage, b)
}
