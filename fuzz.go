// +build gofuzz

package samo

import (
	"encoding/json"
	"fmt"
)

// https://medium.com/@dgryski/go-fuzz-github-com-arolek-ase-3c74d5a3150c
func Fuzz(fdata []byte) int {
	data := fmt.Sprintf("%#v", string(fdata))
	mariadb := &MariaDbStorage{
		User:     "root",
		Password: "",
		Name:     "samo",
		Storage:  &Storage{}}
	leveldb := &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{}}
	memory := &MemoryStorage{
		Memdb:   make(map[string][]byte),
		Storage: &Storage{}}
	tryStore(memory, data)
	tryStore(leveldb, data)
	tryStore(mariadb, data)
	return 1
}

func tryStore(storage Database, data string) {
	storage.Start("/")
	_, err := storage.Set("fuzz", "", 0, data)
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
