package samo

import (
	"errors"
	"os"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDbStorage : composition of storage
type LevelDbStorage struct {
	Path  string
	lvldb *leveldb.DB
	mutex sync.RWMutex
	*Storage
}

// Active  :
func (db *LevelDbStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *LevelDbStorage) Start() error {
	var err error
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	db.lvldb, err = leveldb.OpenFile(db.Path, nil)
	if err == nil {
		db.Storage.Active = true
	}
	return err
}

// Close  :
func (db *LevelDbStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.Storage.Active = false
	db.lvldb.Close()
}

// Clear  :
func (db *LevelDbStorage) Clear() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if !db.Storage.Active {
		os.RemoveAll(db.Path)
	}
}

// Keys  :
func (db *LevelDbStorage) Keys() ([]byte, error) {
	iter := db.lvldb.NewIterator(nil, nil)
	stats := Stats{}
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

	return db.Storage.Objects.encode(stats)
}

// Get  :
func (db *LevelDbStorage) Get(mode string, key string) ([]byte, error) {
	if mode == "sa" {
		data, err := db.lvldb.Get([]byte(key), nil)
		if err != nil {
			return []byte(""), err
		}

		return data, nil
	}

	if mode == "mo" {
		iter := db.lvldb.NewIterator(util.BytesPrefix([]byte(key+db.Storage.Separator)), nil)
		res := []Object{}
		for iter.Next() {
			if db.Storage.Keys.isSub(key, string(iter.Key()), db.Storage.Separator) {
				newObject, err := db.Storage.Objects.decode(iter.Value())
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

		return db.Storage.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Peek :
func (db *LevelDbStorage) Peek(key string, now int64) (int64, int64) {
	previous, err := db.lvldb.Get([]byte(key), nil)
	if err != nil && err.Error() == "leveldb: not found" {
		return now, 0
	}

	if err != nil {
		return now, 0
	}

	oldObject, err := db.Storage.Objects.decode(previous)
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set  :
func (db *LevelDbStorage) Set(key string, index string, now int64, data string) (string, error) {
	created, updated := db.Peek(key, now)
	err := db.lvldb.Put(
		[]byte(key),
		db.Storage.Objects.new(&Object{
			Created: created,
			Updated: updated,
			Index:   index,
			Data:    data,
		}), nil)

	if err != nil {
		return "", err
	}

	return index, nil
}

// Del  :
func (db *LevelDbStorage) Del(key string) error {
	_, err := db.lvldb.Get([]byte(key), nil)
	if err != nil {
		return err
	}

	return db.lvldb.Delete([]byte(key), nil)
}
