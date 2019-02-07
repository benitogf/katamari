package samo

// Database : methods of the persistent data layer
type Database interface {
	Active() bool
	Start(separator string) error
	Close()
	Keys() ([]byte, error)
	Get(mode string, key string) ([]byte, error)
	Set(key string, index string, now int64, data string) (string, error)
	Del(key string) error
	Clear()
}

// Storage : abstraction of persistent data layer
type Storage struct {
	Active    bool
	Separator string
	Db        Database
	*Objects
	*Keys
}
