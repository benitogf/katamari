package katamari

import (
	"github.com/benitogf/katamari/objects"
)

// StorageChan an operation events channel
type StorageChan chan StorageEvent

// StorageEvent an operation event
type StorageEvent struct {
	Key       string
	Operation string
}

// StorageOpt options of the storage instance
type StorageOpt struct {
	NoBroadcastKeys []string
	DbOpt           interface{}
}

// Database interface to be implemented by storages
//
// Active: returns a boolean with the state of the storage
//
// Start: will attempt to start a storage client
//
// Close: closes the storage client
//
// Keys: returns a list with existing keys in the storage
//
// Get(key): retrieve a value or list of values, the key can include a glob pattern
//
// GetObjList(path): retrieve list of values matching a glob pattern without sorting
//
// GetN(path, N): retrieve N list of values matching a glob pattern
//
// Set(key, data): store data under the provided key, key cannot not include glob pattern
//
// Del(key): Delete a key from the storage
//
// Clear: will clear all keys from the storage (used for testing)
//
// Watch: returns a channel that will receive any set or del operation
type Database interface {
	Active() bool
	Start(StorageOpt) error
	Close()
	Keys() ([]byte, error)
	KeysRange(path string, from, to int64) ([]string, error)
	Get(key string) ([]byte, error)
	MemGet(key string) ([]byte, error)
	GetN(path string, limit int) ([]objects.Object, error)
	GetNRange(path string, limit int, from, to int64) ([]objects.Object, error)
	MemGetN(path string, limit int) ([]objects.Object, error)
	GetObjList(path string) ([]objects.Object, error)
	Set(key string, data string) (string, error)
	MemSet(key string, data string) (string, error)
	Pivot(key string, data string, created, updated int64) (string, error)
	Del(key string) error
	MemDel(key string) error
	Clear()
	Watch() StorageChan
	MemWatch() StorageChan
}

// Storage abstraction of persistent data layer
type Storage struct {
	Active bool
	Db     Database
}

// Stats data structure of global keys
type Stats struct {
	Keys []string `json:"keys"`
}

// WatchStorageNoop a noop reader of the watch channel
func WatchStorageNoop(dataStore Database) {
	for {
		_ = <-dataStore.Watch()
		if !dataStore.Active() {
			break
		}
	}
}
