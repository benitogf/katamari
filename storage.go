package samo

// StorageChan : an operation events channel
type StorageChan chan StorageEvent

// StorageEvent : an operation event
type StorageEvent struct {
	key       string
	operation string
}

// Database : methods of the persistent data layer
type Database interface {
	Active() bool
	Start() error
	Close()
	Keys() ([]byte, error)
	Get(mode string, key string) ([]byte, error)
	Set(key string, data string) (string, error)
	Del(key string) error
	Clear()
	Watch() StorageChan
}

// Storage : abstraction of persistent data layer
type Storage struct {
	Active bool
	Db     Database
	*Objects
	*Keys
}

// Stats : data structure of global keys
type Stats struct {
	Keys []string `json:"keys"`
}
