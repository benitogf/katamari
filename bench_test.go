package samo

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// go test -bench -run=^$
func storageSetGetDel(db Database, b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ci, err := db.Set("test/1", "test1")
		require.NoError(b, err)
		_, err = db.Get("sa", "test/"+ci)
		require.NoError(b, err)
		err = db.Del("test/" + ci)
		result, err := db.Get("mo", "test")
		require.NoError(b, err)
		require.Equal(b, "[]", string(result))
	}
}

func BenchmarkMemoryStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b)
}

func BenchmarkLevelStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &LevelStorage{
		Path: "test/db"}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b)
}

func BenchmarkEtcdStorageSetGetDel(b *testing.B) {
	b.ReportAllocs()
	app := Server{}
	app.Silence = true
	app.Storage = &EtcdStorage{OnlyClient: true}
	app.Start("localhost:9889")
	app.Storage.Clear()
	defer app.Close(os.Interrupt)
	storageSetGetDel(app.Storage, b)
}
