package pebble

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
	"github.com/cockroachdb/pebble"
)

// Storage composition of Database interface
type Storage struct {
	Path            string
	mem             sync.Map
	noBroadcastKeys []string
	client          *pebble.DB
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
		db.memWatcher = make(katamari.StorageChan)
	}
	if storageOpt.DbOpt == nil {
		db.client, err = pebble.Open(db.Path, &pebble.Options{})
	} else {
		db.client, err = pebble.Open(db.Path, storageOpt.DbOpt.(*pebble.Options))
	}
	if err == nil {
		db.storage.Active = true
	}
	db.noBroadcastKeys = storageOpt.NoBroadcastKeys
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
	iter := db.client.NewIter(&pebble.IterOptions{})
	iter.First()
	for iter.Valid() {
		_ = db.client.Delete(iter.Key(), pebble.Sync)
		iter.Next()
	}
	iter.Close()
}

// Keys list all the keys in the storage
func (db *Storage) Keys() ([]byte, error) {
	iter := db.client.NewIter(&pebble.IterOptions{})
	stats := katamari.Stats{}

	iter.First()
	for iter.Valid() {
		stats.Keys = append(stats.Keys, string(iter.Key()))
		iter.Next()
	}

	err := iter.Close()
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

	prefixKey := strings.Split(path, "*")[0]
	iter := db.client.NewIter(&pebble.IterOptions{})
	if prefixKey != "" && prefixKey != "*" {
		iter = db.client.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefixKey),
			// UpperBound: []byte(prefixKey + "\x00"),
		})
	}
	iter.First()
	for iter.Valid() {
		if !key.Match(path, string(iter.Key())) {
			iter.Next()
			continue
		}
		current := string(iter.Key())
		paths := strings.Split(current, "/")
		created := key.Decode(paths[len(paths)-1])
		if created < from {
			iter.Next()
			continue
		}
		if created > to {
			iter.Next()
			continue
		}
		keys = append(keys, current)
		iter.Next()
	}

	err := iter.Close()
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

	prefixKey := strings.Split(path, "*")[0]
	iter := db.client.NewIter(&pebble.IterOptions{})
	if prefixKey != "" && prefixKey != "*" {
		iter = db.client.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefixKey),
			// UpperBound: []byte(prefixKey + "\x00"),
		})
	}
	count := 0
	if !iter.Last() {
		err := iter.Close()
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

	err := iter.Close()
	if err != nil {
		return res, err
	}

	return res, nil
}

// GetNRange get last N elements of a pattern related value(s)
func (db *Storage) GetNRange(path string, limit int, from, to int64) ([]objects.Object, error) {
	res := []objects.Object{}
	lookupCount := 0
	lookupLimit := 800000
	if !strings.Contains(path, "*") {
		return res, errors.New("katamari: invalid pattern")
	}

	if limit <= 0 {
		return res, errors.New("katamari: invalid limit")
	}

	prefixKey := strings.Split(path, "*")[0]
	iter := db.client.NewIter(&pebble.IterOptions{})
	if prefixKey != "" && prefixKey != "*" {
		iter = db.client.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefixKey),
			// UpperBound: []byte(prefixKey + "\x00"),
		})
	}
	count := 0
	if !iter.Last() {
		err := iter.Close()
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
			// search limiter
			lookupCount++
			if lookupCount > lookupLimit {
				break
			}
			continue
		}
		if created > to {
			if !iter.Prev() {
				break
			}
			// search limiter
			lookupCount++
			if lookupCount > lookupLimit {
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

	err := iter.Close()
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
		data, closer, err := db.client.Get([]byte(path))
		if err != nil {
			return []byte(""), err
		}

		result := make([]byte, len(data))
		copy(result, data)
		err = closer.Close()
		if err != nil {
			return []byte(""), err
		}
		return result, nil
	}

	prefixKey := strings.Split(path, "*")[0]
	iter := db.client.NewIter(&pebble.IterOptions{})
	if prefixKey != "" && prefixKey != "*" {
		iter = db.client.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefixKey),
			// UpperBound: []byte(prefixKey + "\x00"),
		})
	}
	iter.First()
	res := []objects.Object{}
	for iter.Valid() {
		if !key.Match(path, string(iter.Key())) {
			iter.Next()
			continue
		}

		newObject, err := objects.Decode(iter.Value())
		if err != nil {
			iter.Next()
			continue
		}

		iter.Next()
		res = append(res, newObject)
	}

	err := iter.Close()
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

	prefixKey := strings.Split(path, "*")[0]
	iter := db.client.NewIter(&pebble.IterOptions{})
	if prefixKey != "" && prefixKey != "*" {
		iter = db.client.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefixKey),
			// UpperBound: []byte(prefixKey + "\x00"),
		})
	}
	iter.First()
	for iter.Valid() {
		if !key.Match(path, string(iter.Key())) {
			iter.Next()
			continue
		}

		newObject, err := objects.DecodeFull(iter.Value())
		if err != nil {
			iter.Next()
			continue
		}

		res = append(res, newObject)
		iter.Next()
	}

	return res, iter.Close()
}

// Peek a value timestamps
func (db *Storage) Peek(key string, now int64) (int64, int64) {
	previous, closer, err := db.client.Get([]byte(key))
	if err != nil {
		return now, 0
	}

	oldObject, err := objects.Decode(previous)
	if err != nil {
		return now, 0
	}

	err = closer.Close()
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
	err := db.client.Set(
		[]byte(path),
		objects.New(&objects.Object{
			Created: created,
			Updated: updated,
			Index:   index,
			Data:    data,
		}), pebble.Sync)

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
	err := db.client.Set(
		[]byte(path),
		objects.New(&objects.Object{
			Created: created,
			Updated: updated,
			Index:   index,
			Data:    data,
		}), pebble.Sync)

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
		_, err = db.Get(path)
		if err != nil && err.Error() == "pebble: not found" {
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

	prefixKey := strings.Split(path, "*")[0]
	iter := db.client.NewIter(&pebble.IterOptions{})
	if prefixKey != "" && prefixKey != "*" {
		iter = db.client.NewIter(&pebble.IterOptions{
			LowerBound: []byte(prefixKey),
			// UpperBound: []byte(prefixKey + "\x00"),
		})
	}
	iter.First()
	for iter.Valid() {
		if key.Match(path, string(iter.Key())) {
			err = db.client.Delete(iter.Key(), nil)
			if err != nil {
				break
			}
		}
		iter.Next()
	}
	if err != nil {
		return err
	}

	err = iter.Close()
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
