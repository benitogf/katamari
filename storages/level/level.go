package level

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	katamari "bitbucket.org/idxgames/auth"
	"bitbucket.org/idxgames/auth/key"
	"bitbucket.org/idxgames/auth/objects"
	"github.com/syndtr/goleveldb/leveldb"
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
	memWatcher      katamari.StorageChan
	storage         *katamari.Storage
}

// Active provides access to the status of the storage client
func (db *Storage) Active() bool {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.storage.Active
}

// Start the storage client
func (db *Storage) Start(noBroadcastKeys []string, dbOptions *opt.Options) error {
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
		db.memWatcher = make(katamari.StorageChan)
	}
	if dbOptions == nil {
		db.client, err = leveldb.OpenFile(db.Path, &opt.Options{
			BlockCacheCapacity:     500 * opt.MiB,
			CompactionTableSize:    500 * opt.MiB,
			WriteBuffer:            1024 * opt.MiB,
			CompactionL0Trigger:    40,
			OpenFilesCacheCapacity: 3000,
		})
	} else {
		db.client, err = leveldb.OpenFile(db.Path, dbOptions)
	}
	if err == nil {
		db.storage.Active = true
	}
	db.noBroadcastKeys = noBroadcastKeys
	return err
}

// Close the storage client
func (db *Storage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.storage.Active = false
	db.client.Close()
	close(db.watcher)
	close(db.memWatcher)
	db.watcher = nil
	db.memWatcher = nil
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
	iter := db.client.NewIterator(nil, &opt.ReadOptions{
		DontFillCache: true,
	})
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

// KeysRange list keys in a path and time range
func (db *Storage) KeysRange(path string, from, to int64) ([]string, error) {
	keys := []string{}
	if !strings.Contains(path, "*") {
		return keys, errors.New("katamari: invalid pattern")
	}

	if to < from {
		return keys, errors.New("katamari: invalid range")
	}

	globPrefixKey := strings.Split(path, "*")[0]
	rangeKey := util.BytesPrefix([]byte(globPrefixKey + ""))
	if globPrefixKey == "" || globPrefixKey == "*" {
		rangeKey = nil
	}
	iter := db.client.NewIterator(rangeKey, &opt.ReadOptions{
		DontFillCache: true,
	})

	for iter.Next() {
		if !key.Match(path, string(iter.Key())) {
			continue
		}
		current := string(iter.Key())
		paths := strings.Split(current, "/")
		created := key.Decode(paths[len(paths)-1])
		if created < from {
			continue
		}
		if created > to {
			continue
		}
		keys = append(keys, current)
	}

	iter.Release()
	err := iter.Error()
	if err != nil {
		return keys, err
	}

	return keys, nil
}

// GetN get last N elements of a pattern related value(s)
func (db *Storage) GetN(path string, limit int) ([]objects.Object, error) {
	res := []objects.Object{}
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

	if limit <= 0 {
		return res, errors.New("katamari: invalid limit")
	}

	globPrefixKey := strings.Split(path, "*")[0]
	rangeKey := util.BytesPrefix([]byte(globPrefixKey + ""))
	if globPrefixKey == "" || globPrefixKey == "*" {
		rangeKey = nil
	}
	iter := db.client.NewIterator(rangeKey, &opt.ReadOptions{
		DontFillCache: true,
	})
	count := 0
	if !iter.Last() {
		iter.Release()
		err := iter.Error()
		return res, err
	}
	for count < limit {
		if !key.Match(path, string(iter.Key())) {
			continue
		}

		newObject, err := objects.DecodeFull(iter.Value())
		if err != nil {
			continue
		}

		res = append(res, newObject)
		count++
		if !iter.Prev() {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return res, err
	}

	return res, nil
}

// GetNRange get last N elements of a pattern related value(s)
func (db *Storage) GetNRange(path string, limit int, from, to int64) ([]objects.Object, error) {
	res := []objects.Object{}
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

	if limit <= 0 {
		return res, errors.New("katamari: invalid limit")
	}

	globPrefixKey := strings.Split(path, "*")[0]
	rangeKey := util.BytesPrefix([]byte(globPrefixKey + ""))
	if globPrefixKey == "" || globPrefixKey == "*" {
		rangeKey = nil
	}
	iter := db.client.NewIterator(rangeKey, &opt.ReadOptions{
		DontFillCache: true,
	})
	count := 0
	if !iter.Last() {
		iter.Release()
		err := iter.Error()
		return res, err
	}
	for count < limit {
		if iter.Key() == nil {
			break
		}
		if !key.Match(path, string(iter.Key())) {
			if !iter.Prev() {
				break
			}
			continue
		}

		current := string(iter.Key())
		paths := strings.Split(current, "/")
		created := key.Decode(paths[len(paths)-1])
		if created < from {
			if !iter.Prev() {
				break
			}
			continue
		}
		if created > to {
			if !iter.Prev() {
				break
			}
			continue
		}

		newObject, err := objects.DecodeFull(iter.Value())
		if err != nil {
			if !iter.Prev() {
				break
			}
			continue
		}

		res = append(res, newObject)
		count++
		if !iter.Prev() {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return res, err
	}

	return res, nil
}

// MemGetN get last N elements of a path related value(s)
func (db *Storage) MemGetN(path string, limit int) ([]objects.Object, error) {
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
		if !key.Match(path, string(iter.Key())) {
			continue
		}

		newObject, err := objects.Decode(iter.Value())
		if err != nil {
			continue
		}

		res = append(res, newObject)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return []byte(""), err
	}

	sort.Slice(res, objects.Sort(res))

	return objects.Encode(res)
}

// MemGet a key/pattern related value(s)
func (db *Storage) MemGet(path string) ([]byte, error) {
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
func (db *Storage) GetObjList(path string) ([]objects.Object, error) {
	res := []objects.Object{}
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

	globPrefixKey := strings.Split(path, "*")[0]
	rangeKey := util.BytesPrefix([]byte(globPrefixKey + ""))
	if globPrefixKey == "" || globPrefixKey == "*" {
		rangeKey = nil
	}
	iter := db.client.NewIterator(rangeKey, nil)
	for iter.Next() {
		if !key.Match(path, string(iter.Key())) {
			continue
		}

		newObject, err := objects.DecodeFull(iter.Value())
		if err != nil {
			continue
		}

		res = append(res, newObject)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return res, err
	}

	return res, nil
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

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- katamari.StorageEvent{Key: path, Operation: "set"}
	}
	return index, nil
}

// MemPeek a value timestamps
func (db *Storage) MemPeek(key string, now int64) (int64, int64) {
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

// MemSet a value
func (db *Storage) MemSet(path string, data string) (string, error) {
	now := time.Now().UTC().UnixNano()
	index := key.LastIndex(path)
	created, updated := db.MemPeek(path, now)
	db.mem.Store(path, objects.New(&objects.Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	}))

	db.memWatcher <- katamari.StorageEvent{Key: path, Operation: "set"}
	return index, nil
}

// Pivot set entries on a pivot instance (force created/updated values)
func (db *Storage) Pivot(path string, data string, created int64, updated int64) (string, error) {
	index := key.LastIndex(path)
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

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- katamari.StorageEvent{Key: path, Operation: "set"}
	}
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

		if !key.Contains(db.noBroadcastKeys, path) {
			db.watcher <- katamari.StorageEvent{Key: path, Operation: "del"}
		}
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

	if !key.Contains(db.noBroadcastKeys, path) {
		db.watcher <- katamari.StorageEvent{Key: path, Operation: "del"}
	}
	return nil
}

// MemDel a key/pattern value(s)
func (db *Storage) MemDel(path string) error {
	if !strings.Contains(path, "*") {
		_, found := db.mem.Load(path)
		if !found {
			return errors.New("katamari: not found")
		}
		db.mem.Delete(path)
		db.memWatcher <- katamari.StorageEvent{Key: path, Operation: "del"}
		return nil
	}

	db.mem.Range(func(k interface{}, value interface{}) bool {
		if key.Match(path, k.(string)) {
			db.mem.Delete(k.(string))
		}
		return true
	})
	db.memWatcher <- katamari.StorageEvent{Key: path, Operation: "del"}
	return nil
}

// Watch the storage set/del events
func (db *Storage) Watch() katamari.StorageChan {
	return db.watcher
}

// MemWatch the storage set/del events
func (db *Storage) MemWatch() katamari.StorageChan {
	return db.memWatcher
}
