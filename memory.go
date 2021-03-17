package katamari

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
)

// MemoryStorage composition of Database interface
type MemoryStorage struct {
	mem             sync.Map
	mutex           sync.RWMutex
	noBroadcastKeys []string
	watcher         StorageChan
	storage         *Storage
}

// Active provides access to the status of the storage client
func (db *MemoryStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.storage.Active
}

// Start the storage client
func (db *MemoryStorage) Start(storageOpt StorageOpt) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.storage == nil {
		db.storage = &Storage{}
	}
	if db.watcher == nil {
		db.watcher = make(StorageChan)
	}
	db.noBroadcastKeys = storageOpt.NoBroadcastKeys
	db.storage.Active = true
	return nil
}

// Close the storage client
func (db *MemoryStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	close(db.watcher)
	db.watcher = nil
	db.storage.Active = false
}

// Clear all keys in the storage
func (db *MemoryStorage) Clear() {
	db.mem.Range(func(key interface{}, value interface{}) bool {
		db.mem.Delete(key)
		return true
	})
}

// Keys list all the keys in the storage
func (db *MemoryStorage) Keys() ([]byte, error) {
	stats := Stats{}
	db.mem.Range(func(key interface{}, value interface{}) bool {
		stats.Keys = append(stats.Keys, key.(string))
		return true
	})

	if stats.Keys == nil {
		stats.Keys = []string{}
	}
	sort.Slice(stats.Keys, func(i, j int) bool {
		return strings.ToLower(stats.Keys[i]) < strings.ToLower(stats.Keys[j])
	})

	return objects.Encode(stats)
}

// KeysRange list keys in a path and time range
func (db *MemoryStorage) KeysRange(path string, from, to int64) ([]string, error) {
	keys := []string{}
	if !strings.Contains(path, "*") {
		return keys, errors.New("katamari: invalid pattern")
	}

	if to < from {
		return keys, errors.New("katamari: invalid range")
	}

	db.mem.Range(func(k interface{}, value interface{}) bool {
		current := k.(string)
		if !key.Match(path, current) {
			return true
		}
		paths := strings.Split(current, "/")
		created := key.Decode(paths[len(paths)-1])
		if created < from || created > to {
			return true
		}
		keys = append(keys, current)
		return true
	})

	return keys, nil
}

// Get a key/pattern related value(s)
func (db *MemoryStorage) Get(path string) ([]byte, error) {
	if !strings.Contains(path, "*") {
		data, found := db.mem.Load(path)
		if !found {
			return []byte(""), errors.New("katamari: not found")
		}

		return data.([]byte), nil
	}

	res := []objects.Object{}
	db.mem.Range(func(k interface{}, value interface{}) bool {
		if !key.Match(path, k.(string)) {
			return true
		}

		newObject, err := objects.Decode(value.([]byte))
		if err != nil {
			return true
		}

		res = append(res, newObject)
		return true
	})

	sort.Slice(res, objects.Sort(res))

	return objects.Encode(res)
}

// GetObjList bypass encoding and single objects reads
func (db *MemoryStorage) GetObjList(path string) ([]objects.Object, error) {
	res := []objects.Object{}
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

	db.mem.Range(func(k interface{}, value interface{}) bool {
		if !key.Match(path, k.(string)) {
			return true
		}

		newObject, err := objects.DecodeFull(value.([]byte))
		if err != nil {
			return true
		}

		res = append(res, newObject)
		return true
	})

	return res, nil
}

// GetN get last N elements of a path related value(s)
func (db *MemoryStorage) GetN(path string, limit int) ([]objects.Object, error) {
	res := []objects.Object{}
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

	if limit <= 0 {
		return res, errors.New("katamari: invalid limit")
	}

	db.mem.Range(func(k interface{}, value interface{}) bool {
		if !key.Match(path, k.(string)) {
			return true
		}

		newObject, err := objects.DecodeFull(value.([]byte))
		if err != nil {
			return true
		}

		res = append(res, newObject)
		return true
	})

	sort.Slice(res, objects.Sort(res))

	if len(res) > limit {
		return res[:limit], nil
	}

	return res, nil
}

// GetNRange get last N elements of a path related value(s)
func (db *MemoryStorage) GetNRange(path string, limit int, from, to int64) ([]objects.Object, error) {
	res := []objects.Object{}
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

	if limit <= 0 {
		return res, errors.New("katamari: invalid limit")
	}

	db.mem.Range(func(k interface{}, value interface{}) bool {
		if !key.Match(path, k.(string)) {
			return true
		}

		current := k.(string)
		if !key.Match(path, current) {
			return true
		}
		paths := strings.Split(current, "/")
		created := key.Decode(paths[len(paths)-1])
		if created < from || created > to {
			return true
		}

		newObject, err := objects.DecodeFull(value.([]byte))
		if err != nil {
			return true
		}

		res = append(res, newObject)
		return true
	})

	sort.Slice(res, objects.Sort(res))

	if len(res) > limit {
		return res[:limit], nil
	}

	return res, nil
}

// Peek a value timestamps
func (db *MemoryStorage) Peek(key string, now int64) (int64, int64) {
	previous, found := db.mem.Load(key)
	if !found {
		return now, 0
	}

	oldObject, err := objects.Decode(previous.([]byte))
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set a value
func (db *MemoryStorage) Set(path string, data string) (string, error) {
	now := time.Now().UTC().UnixNano()
	index := key.LastIndex(path)
	created, updated := db.Peek(path, now)
	db.mem.Store(path, objects.New(&objects.Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	}))

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- StorageEvent{Key: path, Operation: "set"}
	}
	return index, nil
}

// Pivot set entries on pivot instances (force created/updated values)
func (db *MemoryStorage) Pivot(path string, data string, created int64, updated int64) (string, error) {
	index := key.LastIndex(path)
	db.mem.Store(path, objects.New(&objects.Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	}))

	if len(path) > 8 && path[0:7] == "history" {
		return index, nil
	}

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- StorageEvent{Key: path, Operation: "set"}
	}
	return index, nil
}

// Del a key/pattern value(s)
func (db *MemoryStorage) Del(path string) error {
	if !strings.Contains(path, "*") {
		_, found := db.mem.Load(path)
		if !found {
			return errors.New("katamari: not found")
		}
		db.mem.Delete(path)
		if !key.Contains(db.noBroadcastKeys, path) {
			db.watcher <- StorageEvent{Key: path, Operation: "del"}
		}
		return nil
	}

	db.mem.Range(func(k interface{}, value interface{}) bool {
		if key.Match(path, k.(string)) {
			db.mem.Delete(k.(string))
		}
		return true
	})
	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- StorageEvent{Key: path, Operation: "del"}
	}
	return nil
}

// Watch the storage set/del events
func (db *MemoryStorage) Watch() StorageChan {
	return db.watcher
}
