package samo

import (
	"bytes"
	"encoding/json"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Storage : abstraction of persistent data layer
type Storage struct {
	kind      string
	lvldb     *leveldb.DB
	path      string
	active    bool
	console   *Console
	helpers   *Helpers
	separator string
}

func (db *Storage) start(console *Console, separator string) error {
	var err error
	db.console = console
	db.separator = separator
	db.helpers = &Helpers{}
	db.lvldb, err = leveldb.OpenFile(db.path, nil)
	if err == nil {
		db.active = true
	}
	return err
}

func (db *Storage) close() {
	if db.lvldb != nil {
		db.active = false
		db.lvldb.Close()
	}
}

func (db *Storage) getKeys() ([]byte, error) {
	iter := db.lvldb.NewIterator(nil, nil)
	stats := Stats{}
	for iter.Next() {
		stats.Keys = append(stats.Keys, string(iter.Key()))
	}
	iter.Release()
	err := iter.Error()
	if err == nil {
		if stats.Keys == nil {
			stats.Keys = []string{}
		}
		respJSON, err := json.Marshal(stats)
		if err == nil {
			return respJSON, nil
		}
	}

	return nil, err
}

func (db *Storage) getData(mode string, key string) []byte {
	switch mode {
	case "sa":
		data, err := db.lvldb.Get([]byte(key), nil)
		if err == nil {
			return data
		}
		db.console.err("getError["+mode+"/"+key+"]", err)
	case "mo":
		iter := db.lvldb.NewIterator(util.BytesPrefix([]byte(key+db.separator)), nil)
		res := []Object{}
		for iter.Next() {
			if db.helpers.isMO(key, string(iter.Key()), db.separator) {
				var newObject Object
				err := json.Unmarshal(iter.Value(), &newObject)
				if err == nil {
					res = append(res, newObject)
				} else {
					db.console.err("getError["+mode+"/"+key+"]", err)
				}
			}
		}
		iter.Release()
		err := iter.Error()
		if err == nil {
			data, err := json.Marshal(res)
			if err == nil {
				return data
			}
		}
	}

	return []byte("")
}

func (db *Storage) setData(mode string, key string, index string, now int64, data string) (string, error) {
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
	if err == nil {
		db.console.log("set[" + mode + "/" + key + "]")
		return index, nil
	}

	db.console.err("setError["+mode+"/"+key+"]", err)
	return "", err
}

func (db *Storage) delData(mode string, key string, index string) error {
	if mode == "mo" {
		key += db.separator + index
	}

	_, err := db.lvldb.Get([]byte(key), nil)
	if err == nil {
		err := db.lvldb.Delete([]byte(key), nil)
		if err == nil {
			db.console.log("delete[" + mode + "/" + key + "]")
			return nil
		}
	}

	db.console.err("deleteError["+mode+"/"+key+"]", err)
	return err
}
