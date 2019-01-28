package samo

import (
	"bytes"
	"encoding/json"
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
	db.Lock = sync.RWMutex{}
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

	resp, err := json.Marshal(stats)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Get :
func (db *MemoryStorage) Get(mode string, key string) ([]byte, error) {
	db.Lock.RLock()
	defer db.Lock.RUnlock()
	var err error
	switch mode {
	case "sa":
		data := db.Memdb[key]
		if data == nil {
			return []byte(""), err
		}

		return data, nil
	case "mo":
		res := []Object{}
		for k := range db.Memdb {
			if (&Helpers{}).IsMO(key, k, db.Storage.Separator) {
				var newObject Object
				err := json.Unmarshal(db.Memdb[k], &newObject)
				if err == nil {
					res = append(res, newObject)
				}
			}
		}

		data, err := json.Marshal(res)
		if err != nil {
			return []byte(""), err
		}

		return data, nil
	default:
		return []byte(""), errors.New("SAMO: unrecognized mode: " + mode)
	}
}

// Set  :
func (db *MemoryStorage) Set(key string, index string, now int64, data string) (string, error) {
	db.Lock.Lock()
	defer db.Lock.Unlock()
	updated := now
	created := now
	previous := db.Memdb[key]
	if previous == nil {
		updated = 0
	} else {
		var oldObject Object
		err := json.Unmarshal(previous, &oldObject)
		if err == nil {
			created = oldObject.Created
		}
	}

	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	})

	db.Memdb[key] = dataBytes.Bytes()
	return index, nil
}

// Del  :
func (db *MemoryStorage) Del(key string) error {
	db.Lock.Lock()
	defer db.Lock.Unlock()
	if db.Memdb[key] == nil {
		return errors.New("SAMO: not found")
	}
	delete(db.Memdb, key)
	return nil
}
