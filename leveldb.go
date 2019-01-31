package samo

import (
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDbStorage : composition of storage
type LevelDbStorage struct {
	Path  string
	lvldb *leveldb.DB
	*Objects
	*Storage
}

// Active  :
func (db *LevelDbStorage) Active() bool {
	return db.Storage.Active
}

// Start  :
func (db *LevelDbStorage) Start(separator string) error {
	var err error
	db.Storage.Separator = separator
	db.Objects = &Objects{&Keys{}}
	db.lvldb, err = leveldb.OpenFile(db.Path, nil)
	if err == nil {
		db.Storage.Active = true
	}
	return err
}

// Close  :
func (db *LevelDbStorage) Close() {
	if db.lvldb != nil {
		db.Storage.Active = false
		db.lvldb.Close()
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

	return db.Objects.encode(stats)
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
			if db.Objects.Keys.isSub(key, string(iter.Key()), db.Storage.Separator) {
				newObject, err := db.Objects.read(iter.Value())
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

		return db.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Peek :
func (db *LevelDbStorage) Peek(key string, now int64) (int64, int64) {
	updated := now
	created := now
	previous, err := db.lvldb.Get([]byte(key), nil)
	if err != nil && err.Error() == "leveldb: not found" {
		updated = 0
	}

	if err == nil {
		oldObject, err := db.Objects.read(previous)
		if err == nil {
			created = oldObject.Created
		}
	}

	return updated, created
}

// Set  :
func (db *LevelDbStorage) Set(key string, index string, now int64, data string) (string, error) {
	updated, created := db.Peek(key, now)
	err := db.lvldb.Put(
		[]byte(key),
		db.Objects.write(&Object{
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
