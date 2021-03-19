package lvlmap

import (
	"errors"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
	"github.com/syndtr/goleveldb/leveldb"
	errorsLeveldb "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Storage composition of Database interface
type Storage struct {
	Path            string
	mem             sync.Map
	noBroadcastKeys []string
	client          *leveldb.DB
	mutex           sync.RWMutex
	watcher         katamari.StorageChan
	storage         *katamari.Storage
}

// Active provides access to the status of the storage client
func (db *Storage) Active() bool {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.storage.Active
}

func (db *Storage) recover() error {
	var err error
	db.client, err = leveldb.RecoverFile(db.Path, &opt.Options{})

	// recover failed
	if err != nil {
		return err
	}

	return nil
}

// Start the storage client
func (db *Storage) Start(storageOpt katamari.StorageOpt) error {
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

	db.client, err = leveldb.OpenFile(db.Path, &opt.Options{})

	if errorsLeveldb.IsCorrupted(err) {
		log.Println("db is corrupted, attempting recover", err)
		err = db.recover()
		if err != nil {
			log.Println("failed to recover db", err)
			return err
		}
	}

	// load db snapshot into db
	err = db.load()
	if err != nil {
		log.Println("failed to load db snapshot", err)
		return err
	}

	db.storage.Active = true
	db.noBroadcastKeys = storageOpt.NoBroadcastKeys
	return nil
}

func (db *Storage) load() error {
	iter := db.client.NewIterator(nil, &opt.ReadOptions{
		DontFillCache: true,
	})

	for iter.Next() {
		path := string(iter.Key())
		value := iter.Value()
		tmp := make([]byte, len(value))
		copy(tmp, value)
		db.mem.Store(path, tmp)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return err
	}

	return nil
}

// Close the storage client
func (db *Storage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.storage.Active = false
	db.client.Close()
	close(db.watcher)
	db.watcher = nil
}

// Clear all keys in the storage
func (db *Storage) Clear() {
	db.mem.Range(func(key interface{}, value interface{}) bool {
		db.mem.Delete(key)
		return true
	})
	iter := db.client.NewIterator(nil, nil)
	for iter.Next() {
		_ = db.client.Delete(iter.Key(), nil)
	}
	iter.Release()
}

// Keys list all the keys in the storage
func (db *Storage) Keys() ([]byte, error) {
	stats := katamari.Stats{}
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
func (db *Storage) KeysRange(path string, from, to int64) ([]string, error) {
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

// GetNRange get last N elements of a path related value(s) for a given time range
func (db *Storage) GetNRange(path string, limit int, from, to int64) ([]objects.Object, error) {
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

		newObject, err := objects.Decode(value.([]byte))
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

// GetN get last N elements of a path related value(s)
func (db *Storage) GetN(path string, limit int) ([]objects.Object, error) {
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

		newObject, err := objects.Decode(value.([]byte))
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

// Get a key/pattern related value(s)
func (db *Storage) Get(path string) ([]byte, error) {
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

		newObject, err := objects.DecodeRaw(value.([]byte))
		if err != nil {
			return true
		}

		res = append(res, newObject)
		return true
	})

	sort.Slice(res, objects.Sort(res))

	return objects.Encode(res)
}

// GetObjList force base64 decoding and bypass sorting
func (db *Storage) GetObjList(path string) ([]objects.Object, error) {
	res := []objects.Object{}
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

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

	return res, nil
}

// Peek a value timestamps
func (db *Storage) Peek(key string, now int64) (int64, int64) {
	previous, found := db.mem.Load(key)
	if !found {
		return now, 0
	}

	oldObject, err := objects.DecodeRaw(previous.([]byte))
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

	aux := objects.New(&objects.Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	})

	db.mem.Store(path, aux)
	err := db.client.Put([]byte(path), aux, nil)

	if err != nil {
		return "", err
	}

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- katamari.StorageEvent{Key: path, Operation: "set"}
	}

	return index, nil
}

// Pivot set entries on a pivot instance (force created/updated values)
func (db *Storage) Pivot(path string, data string, created int64, updated int64) (string, error) {
	index := key.LastIndex(path)
	aux := objects.New(&objects.Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	})

	db.mem.Store(path, aux)
	err := db.client.Put([]byte(path), aux, nil)

	if err != nil {
		return "", err
	}

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- katamari.StorageEvent{Key: path, Operation: "set"}
	}

	return index, nil
}

// Del a key/pattern value(s)
func (db *Storage) Del(path string) error {
	var err error
	if !strings.Contains(path, "*") {
		_, found := db.mem.Load(path)
		if !found {
			return errors.New("katamari: not found")
		}
		db.mem.Delete(path)

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

		if !key.Contains(db.noBroadcastKeys, path) {
			db.watcher <- katamari.StorageEvent{Key: path, Operation: "del"}
		}
		return nil
	}

	db.mem.Range(func(k interface{}, value interface{}) bool {
		if key.Match(path, k.(string)) {
			db.mem.Delete(k.(string))
		}
		return true
	})

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

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- katamari.StorageEvent{Key: path, Operation: "del"}
	}
	return nil
}

// Watch the storage set/del events
func (db *Storage) Watch() katamari.StorageChan {
	return db.watcher
}
