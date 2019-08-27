package katamari

// StorageChan an operation events channel
type StorageChan chan StorageEvent

// StorageEvent an operation event
type StorageEvent struct {
	Key       string
	Operation string
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
// Set(key, data): store data under the provided key, key cannot not include glob pattern
//
// Del(key): Delete a key from the storage
//
// Clear: will clear all keys from the storage (used for testing)
//
// Watch: returns a channel that will receive any set or del operation
type Database interface {
	Active() bool
	Start() error
	Close()
	Keys() ([]byte, error)
	Get(key string) ([]byte, error)
	Set(key string, data string) (string, error)
	Del(key string) error
	Clear()
	Watch() StorageChan
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
