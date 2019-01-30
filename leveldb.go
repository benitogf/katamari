package samo

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDbStorage : composition of storage
type LevelDbStorage struct {
	Path  string
	lvldb *leveldb.DB
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
	resp, err := json.Marshal(stats)
	if err != nil {
		return nil, err
	}

	return resp, nil
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
			if (&Helpers{}).IsMO(key, string(iter.Key()), db.Storage.Separator) {
				var newObject Object
				err := json.Unmarshal(iter.Value(), &newObject)
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

		data, err := json.Marshal(res)
		if err != nil {
			return []byte(""), err
		}

		return data, nil
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Set  :
func (db *LevelDbStorage) Set(key string, index string, now int64, data string) (string, error) {
	updated := now
	created := now
	previous, err := db.lvldb.Get([]byte(key), nil)
	if err != nil && err.Error() == "leveldb: not found" {
		updated = 0
	}

	if err == nil {
		var oldObject Object
		err = json.Unmarshal(previous, &oldObject)
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

	err = db.lvldb.Put(
		[]byte(key),
		dataBytes.Bytes(), nil)

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
