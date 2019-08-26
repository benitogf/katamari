// +build gofuzz

package katamari

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// https://medium.com/@dgryski/go-fuzz-github-com-arolek-ase-3c74d5a3150c
// go get -u github.com/dvyukov/go-fuzz/go-fuzz-build
// go get -u github.com/dvyukov/go-fuzz/go-fuzz
// go-fuzz-build github.com/benitogf/katamari
// go-fuzz -bin='katamari-fuzz.zip' -workdir=fuzz
func Fuzz(fdata []byte) int {
	data := fmt.Sprintf("%#v", string(fdata))
	memory := &MemoryStorage{}
	level := &LevelDbStorage{
		Path:  "test/db",
		lvldb: nil}
	redis := &RedisStorage{
		Address:  "localhost:6379",
		Password: ""}
	etcd := &EtcdStorage{}
	mongo := &MongodbStorage{
		Address: "localhost:27017"}
	fuzzStorage(memory, data)
	fuzzStorage(level, data)
	fuzzStorage(etcd, data)
	return 1
}

func fuzzStorage(storage Database, data string) {
	storage.Start()
	_, err := storage.Set("fuzz", "", 0, base64.StdEncoding.EncodeToString([]byte(data)))
	if err != nil {
		storage.Close()
		panic(err)
	}
	raw, err := storage.Get("sa", "fuzz")
	if err != nil {
		storage.Close()
		panic(err)
	}
	var obj Object
	err = json.Unmarshal(raw, &obj)
	if err != nil {
		storage.Close()
		panic(err)
	}
	raw, err = base64.StdEncoding.DecodeString(obj.Data)
	if err != nil {
		panic(err)
	}
	obj.Data = string(raw)
	if obj.Data != string(data) {
		panic("data != obj.Data: " + obj.Data + " : " + data)
	}
	err = storage.Del("fuzz")
	if err != nil {
		storage.Close()
		panic(err)
	}
	post, err := storage.Get("sa", "fuzz")
	if err == nil {
		storage.Close()
		panic("expected empty but got: " + string(post))
	}
	storage.Close()
}
