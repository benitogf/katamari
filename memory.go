package samo

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

// MemoryStorage : composition of storage
type MemoryStorage struct {
	Memdb map[string][]byte
	Lock  sync.RWMutex
	*Storage
}

// Active  :
func (db *MemoryStorage) Active() bool {
	return db.Storage.Active
}

// Start  :
func (db *MemoryStorage) Start(separator string) error {
	db.Storage.Separator = separator
	db.Storage.Active = true
	return nil
}

// Close  :
func (db *MemoryStorage) Close() {
	db.Storage.Active = false
}

// Keys  :
func (db *MemoryStorage) Keys() ([]byte, error) {
	db.Lock.RLock()
	defer db.Lock.RUnlock()
	stats := Stats{}
	for k := range db.Memdb {
		stats.Keys = append(stats.Keys, k)
	}

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
	db.Lock.RLock()
	defer db.Lock.RUnlock()
	if mode == "sa" {
		data := db.Memdb[key]
		if data == nil {
			return []byte(""), errors.New("samo: not found")
		}

		return data, nil
	}

	if mode == "mo" {
		res := []Object{}
		for k := range db.Memdb {
			if db.Storage.Keys.isSub(key, k, db.Storage.Separator) {
				newObject, err := db.Storage.Objects.read(db.Memdb[k])
				if err == nil {
					res = append(res, newObject)
				}
			}
		}

		return db.Storage.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Peek will check the object stored in the key if any, returns created and updated times acordingly
func (db *MemoryStorage) Peek(key string, now int64) (int64, int64) {
	previous := db.Memdb[key]
	if previous == nil {
		return now, 0
	}

	oldObject, err := db.Storage.Objects.read(previous)
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set  :
func (db *MemoryStorage) Set(key string, index string, now int64, data string) (string, error) {
	db.Lock.Lock()
	defer db.Lock.Unlock()
	created, updated := db.Peek(key, now)
	db.Memdb[key] = db.Storage.Objects.write(&Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	})
	return index, nil
}

// Del  :
func (db *MemoryStorage) Del(key string) error {
	db.Lock.Lock()
	defer db.Lock.Unlock()
	if db.Memdb[key] == nil {
		return errors.New("samo: not found")
	}
	delete(db.Memdb, key)
	return nil
}
