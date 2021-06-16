package stream

import (
	"time"
)

// Cache holds version and data
type Cache struct {
	Version int64
	Data    []byte
}

// _setCache will store data in a pool's cache
func (sm *Pools) setCache(poolIndex int, data []byte) int64 {
	now := time.Now().UTC().UnixNano()
	sm.Pools[poolIndex].cache = Cache{
		Version: now,
		Data:    data,
	}
	return now
}

// GetCache will get a pool's cache
func (sm *Pools) getCache(poolIndex int) Cache {
	return sm.Pools[poolIndex].cache
}
