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

// SetCache will store data in a pool's cache
func (sm *Pools) SetCache(poolIndex int, data []byte) int64 {
	sm.mutex.Lock()
	now := time.Now().UTC().UnixNano()
	sm.Pools[poolIndex].cache = Cache{
		Version: now,
		Data:    data,
	}
	sm.mutex.Unlock()
	return now
}

// GetCache will get a pool's cache
func (sm *Pools) GetCache(poolIndex int) Cache {
	sm.mutex.RLock()
	cache := sm.Pools[poolIndex].cache
	sm.mutex.RUnlock()
	return cache
}

// SetPoolCache will store data in a pool's cache
func (sm *Pools) SetPoolCache(key string, data []byte) int64 {
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

// GetPoolCache will get a pool's cache
func (sm *Pools) GetPoolCache(key string) (Cache, error) {
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
