package level

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Storage composition of Database interface
type Storage struct {
	Path    string
	client  *leveldb.DB
	mutex   sync.RWMutex
	watcher katamari.StorageChan
	storage *katamari.Storage
}

// Active provides access to the status of the storage client
func (db *Storage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.storage.Active
}

// Start the storage client
func (db *Storage) Start() error {
	var err error
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.storage == nil {
		db.storage = &katamari.Storage{}
	}
	if db.Path == "" {
		db.Path = "data/db"
	}
	if db.watcher == nil {
		db.watcher = make(katamari.StorageChan)
	}
	db.client, err = leveldb.OpenFile(db.Path, nil)
	if err == nil {
		db.storage.Active = true
	}
	return err
}

// Close the storage client
func (db *Storage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.storage.Active = false
	db.client.Close()
	close(db.watcher)
}

// Clear all keys in the storage
func (db *Storage) Clear() {
	iter := db.client.NewIterator(nil, nil)
	for iter.Next() {
		_ = db.client.Delete(iter.Key(), nil)
	}
	iter.Release()
}

// Keys list all the keys in the storage
func (db *Storage) Keys() ([]byte, error) {
	iter := db.client.NewIterator(nil, nil)
	stats := katamari.Stats{}
	for iter.Next() {
		stats.Keys = append(stats.Keys, string(iter.Key()))
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}

	if stats.Keys == nil {
		stats.Keys = []string{}
	}

	return objects.Encode(stats)
}

// Get a key/pattern related value(s)
func (db *Storage) Get(path string) ([]byte, error) {
	if !strings.Contains(path, "*") {
		data, err := db.client.Get([]byte(path), nil)
		if err != nil {
			return []byte(""), err
		}

		return data, nil
	}

	globPrefixKey := strings.Split(path, "*")[0]
	rangeKey := util.BytesPrefix([]byte(globPrefixKey + ""))
	if globPrefixKey == "" || globPrefixKey == "*" {
		rangeKey = nil
	}
	iter := db.client.NewIterator(rangeKey, nil)
	res := []objects.Object{}
	for iter.Next() {
		if key.Match(path, string(iter.Key())) {
			newObject, err := objects.Decode(iter.Value())
			if err == nil {
				res = append(res, newObject)
			}
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return []byte(""), err
	}

	sort.Slice(res, objects.Sort(res))

	return objects.Encode(res)
}

// Peek a value timestamps
func (db *Storage) Peek(key string, now int64) (int64, int64) {
	previous, err := db.client.Get([]byte(key), nil)
	if err != nil {
		return now, 0
	}

	oldObject, err := objects.Decode(previous)
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set a value
func (db *Storage) Set(path string, data string) (string, error) {
	now := time.Now().UTC().UnixNano()
	index := key.LastIndex(path)
	created, updated := db.Peek(path, now)
	err := db.client.Put(
		[]byte(path),
		objects.New(&objects.Object{
			Created: created,
			Updated: updated,
			Index:   index,
			Data:    data,
		}), nil)

	if err != nil {
		return "", err
	}

	db.watcher <- katamari.StorageEvent{Key: path, Operation: "set"}
	return index, nil
}

// Del a key/pattern value(s)
func (db *Storage) Del(path string) error {
	var err error
	if !strings.Contains(path, "*") {
		_, err = db.client.Get([]byte(path), nil)
		if err != nil && err.Error() == "leveldb: not found" {
			return errors.New("katamari: not found")
		}

		if err != nil {
			return err
		}

		err = db.client.Delete([]byte(path), nil)
		if err != nil {
			return err
		}
		db.watcher <- katamari.StorageEvent{Key: path, Operation: "del"}
		return nil
	}

	globPrefixKey := strings.Split(path, "*")[0]
	rangeKey := util.BytesPrefix([]byte(globPrefixKey + ""))
	if globPrefixKey == "" || globPrefixKey == "*" {
		rangeKey = nil
	}
	iter := db.client.NewIterator(rangeKey, nil)
	for iter.Next() {
		if key.Match(path, string(iter.Key())) {
			err = db.client.Delete(iter.Key(), nil)
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		return err
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		return err
	}

	db.watcher <- katamari.StorageEvent{Key: path, Operation: "del"}
	return nil
}

// Watch the storage set/del events
func (db *Storage) Watch() katamari.StorageChan {
	return db.watcher
}
