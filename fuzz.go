// +build gofuzz

package samo

import "encoding/json"

// https://medium.com/@dgryski/go-fuzz-github-com-arolek-ase-3c74d5a3150c
func Fuzz(fdata []byte) int {
	data := (&Messages{}).write(fdata)
	if data == "" {
		return 0
	}
	mariadb := &MariaDbStorage{
		User:     "root",
		Password: "",
		Name:     "samo",
		Storage:  &Storage{Active: false}}
	leveldb := &LevelDbStorage{
		Path:    "test/db",
		lvldb:   nil,
		Storage: &Storage{Active: false}}
	memory := &MemoryStorage{
		Memdb:   make(map[string][]byte),
		Storage: &Storage{Active: false}}
	tryStore(memory, data)
	tryStore(leveldb, data)
	tryStore(mariadb, data)
	return 1
}

func tryStore(storage Database, data string) {
	storage.Start("/")
	defer storage.Close()
	_, err := storage.Set("fuzz", "", 0, data)
	if err != nil {
		panic(err)
	}
	raw, err := storage.Get("sa", "fuzz")
	if err != nil {
		panic(err)
	}
	var obj Object
	err = json.Unmarshal(raw, &obj)
	if err != nil {
		panic(err)
	}
	if obj.Data != string(data) {
		panic("data != obj.Data: " + obj.Data + " : " + data)
	}
	err = storage.Del("fuzz")
	if err != nil {
		panic(err)
	}
	post, err := storage.Get("sa", "fuzz")
	if err == nil {
		panic("expected empty but got: " + string(post))
	}
}
