package samo

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

// MemoryStorage : composition of storage
type MemoryStorage struct {
	Memdb sync.Map
	mutex sync.RWMutex
	// watcher StorageChan
	*Storage
}

// Active  :
func (db *MemoryStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *MemoryStorage) Start() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.Storage == nil {
		db.Storage = &Storage{}
	}
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	// if db.watcher == nil {
	// 	db.watcher = make(StorageChan)
	// }
	db.Storage.Active = true
	return nil
}

// Close  :
func (db *MemoryStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	// close(db.watcher)
	db.Storage.Active = false
}

// Clear  :
func (db *MemoryStorage) Clear() {
	db.Memdb.Range(func(key interface{}, value interface{}) bool {
		db.Memdb.Delete(key)
		return true
	})
}

// Keys  :
func (db *MemoryStorage) Keys() ([]byte, error) {
	stats := Stats{}
	db.Memdb.Range(func(key interface{}, value interface{}) bool {
		stats.Keys = append(stats.Keys, key.(string))
		return true
	})

	if stats.Keys == nil {
		stats.Keys = []string{}
	}
	sort.Slice(stats.Keys, func(i, j int) bool {
		return strings.ToLower(stats.Keys[i]) < strings.ToLower(stats.Keys[j])
	})

	return db.Storage.Objects.encode(stats)
}

// Get :
func (db *MemoryStorage) Get(mode string, key string) ([]byte, error) {
	if mode == "sa" {
		data, found := db.Memdb.Load(key)
		if !found {
			return []byte(""), errors.New("samo: not found")
		}

		return data.([]byte), nil
	}

	if mode == "mo" {
		res := []Object{}
		db.Memdb.Range(func(k interface{}, value interface{}) bool {
			if db.Storage.Keys.isSub(key, k.(string), db.Storage.Separator) {
				newObject, err := db.Storage.Objects.decode(value.([]byte))
				if err == nil {
					res = append(res, newObject)
				}
			}
			return true
		})

		return db.Storage.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Peek will check the object stored in the key if any, returns created and updated times accordingly
func (db *MemoryStorage) Peek(key string, now int64) (int64, int64) {
	previous, found := db.Memdb.Load(key)
	if !found {
		return now, 0
	}

	oldObject, err := db.Storage.Objects.decode(previous.([]byte))
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set  :
func (db *MemoryStorage) Set(key string, index string, now int64, data string) (string, error) {
	created, updated := db.Peek(key, now)
	db.Memdb.Store(key, db.Storage.Objects.new(&Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	}))
	// db.watcher <- StorageEvent{key: key, operation: "set"}
	return index, nil
}

// Del  :
func (db *MemoryStorage) Del(key string) error {
	_, found := db.Memdb.Load(key)
	if !found {
		return errors.New("samo: not found")
	}
	db.Memdb.Delete(key)
	// db.watcher <- StorageEvent{key: key, operation: "del"}
	return nil
}

// Watch :
func (db *MemoryStorage) Watch() StorageChan {
	// return db.watcher
	return nil
}
