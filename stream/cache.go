package stream

import (
	"errors"
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

// SetCache by key
func (sm *Pools) SetCache(key string, data []byte) int64 {
	sm.mutex.Lock()
	poolIndex := sm.findPool(key, key)
	now := time.Now().UTC().UnixNano()
	if poolIndex == -1 {
		// create a pool
		sm.Pools = append(
			sm.Pools,
			&Pool{
				Key: key,
				cache: Cache{
					Version: now,
					Data:    data,
				},
				connections: []*Conn{}})
		sm.mutex.Unlock()
		return now
	}
	sm.Pools[poolIndex].cache = Cache{
		Version: now,
		Data:    data,
	}
	sm.mutex.Unlock()

	return now
}

// GetCache by key
func (sm *Pools) GetCache(key string) (Cache, error) {
	sm.mutex.RLock()
	poolIndex := sm.findPool(key, key)
	if poolIndex == -1 {
		sm.mutex.RUnlock()
		return Cache{}, errors.New("stream pool not found")
	}
	cache := sm.Pools[poolIndex].cache
	if len(cache.Data) == 0 {
		sm.mutex.RUnlock()
		return cache, errors.New("stream pool cache empty")
	}
	sm.mutex.RUnlock()
	return cache, nil
}
