package katamari

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStorage composition of Database interface
type MemoryStorage struct {
	Memdb   sync.Map
	mutex   sync.RWMutex
	watcher StorageChan
	storage *Storage
}

// Active provides access to the status of the storage client
func (db *MemoryStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.storage.Active
}

// Start the storage client
func (db *MemoryStorage) Start() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.storage == nil {
		db.storage = &Storage{}
	}
	if db.watcher == nil {
		db.watcher = make(StorageChan)
	}
	db.storage.Active = true
	return nil
}

// Close the storage client
func (db *MemoryStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	close(db.watcher)
	db.storage.Active = false
}

// Clear all keys in the storage
func (db *MemoryStorage) Clear() {
	db.Memdb.Range(func(key interface{}, value interface{}) bool {
		db.Memdb.Delete(key)
		return true
	})
}

// Keys list all the keys in the storage
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

	return db.storage.Objects.Encode(stats)
}

// Get a key/pattern related value(s)
func (db *MemoryStorage) Get(key string) ([]byte, error) {
	if !strings.Contains(key, "*") {
		data, found := db.Memdb.Load(key)
		if !found {
			return []byte(""), errors.New("katamari: not found")
		}

		return data.([]byte), nil
	}

	res := []Object{}
	db.Memdb.Range(func(k interface{}, value interface{}) bool {
		if db.storage.Keys.Match(key, k.(string)) {
			newObject, err := db.storage.Objects.Decode(value.([]byte))
			if err == nil {
				res = append(res, newObject)
			}
		}
		return true
	})

	sort.Slice(res, db.storage.Objects.Sort(res))

	return db.storage.Objects.Encode(res)
}

// Peek a value timestamps
func (db *MemoryStorage) Peek(key string, now int64) (int64, int64) {
	previous, found := db.Memdb.Load(key)
	if !found {
		return now, 0
	}

	oldObject, err := db.storage.Objects.Decode(previous.([]byte))
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set a value
func (db *MemoryStorage) Set(key string, data string) (string, error) {
	now := time.Now().UTC().UnixNano()
	index := (&Keys{}).LastIndex(key)
	created, updated := db.Peek(key, now)
	db.Memdb.Store(key, db.storage.Objects.New(&Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	}))
	db.watcher <- StorageEvent{Key: key, Operation: "set"}
	return index, nil
}

// Del a key/pattern value(s)
func (db *MemoryStorage) Del(key string) error {
	if !strings.Contains(key, "*") {
		_, found := db.Memdb.Load(key)
		if !found {
			return errors.New("katamari: not found")
		}
		db.Memdb.Delete(key)
		db.watcher <- StorageEvent{Key: key, Operation: "del"}
		return nil
	}

	db.Memdb.Range(func(k interface{}, value interface{}) bool {
		if db.storage.Keys.Match(key, k.(string)) {
			db.Memdb.Delete(k.(string))
		}
		return true
	})
	db.watcher <- StorageEvent{Key: key, Operation: "del"}
	return nil
}

// Watch the storage set/del events
func (db *MemoryStorage) Watch() StorageChan {
	return db.watcher
}
