package samo

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/go-redis/redis"
)

type RedisStorage struct {
	mutex    sync.RWMutex
	address  string
	password string
	redisdb  *redis.Client
	*Storage
}

// Active  :
func (db *RedisStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *RedisStorage) Start() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	db.Storage.Active = true
	if db.address == "" {
		db.address = "localhost:6379"
	}
	db.redisdb = redis.NewClient(&redis.Options{
		Addr:     db.address,
		Password: db.password, // no password set
		DB:       0,           // use default DB
	})
	_, err := db.redisdb.Ping().Result()
	return err
}

// Close  :
func (db *RedisStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.redisdb.Close()
	db.Storage.Active = false
}

// Clear :
func (db *RedisStorage) Clear() {
	db.redisdb.FlushDB()
}

// Keys  :
func (db *RedisStorage) Keys() ([]byte, error) {
	stats := Stats{}
	var cursor uint64
	for {
		var err error
		stats.Keys, cursor, err = db.redisdb.Scan(cursor, "*", -1).Result()
		if err != nil {
			return nil, err
		}
		if cursor == 0 {
			break
		}
	}

	sort.Slice(stats.Keys, func(i, j int) bool {
		return strings.ToLower(stats.Keys[i]) < strings.ToLower(stats.Keys[j])
	})

	return db.Storage.Objects.encode(stats)
}

// Get :
func (db *RedisStorage) Get(mode string, key string) ([]byte, error) {
	if mode == "sa" {
		data, err := db.redisdb.Get(key).Result()
		if err == redis.Nil {
			return []byte(""), errors.New("samo: not found")
		}
		if err != nil {
			return []byte(""), err
		}

		return []byte(data), nil
	}

	if mode == "mo" {
		res := []Object{}
		var keys []string
		var cursor uint64
		for {
			var err error
			keys, cursor, err = db.redisdb.Scan(cursor, key+db.Separator+"*", -1).Result()
			if err != nil {
				return nil, err
			}
			if cursor == 0 {
				break
			}
		}
		for i := range keys {
			if db.Storage.Keys.isSub(key, keys[i], db.Storage.Separator) {
				data, err := db.redisdb.Get(keys[i]).Result()
				if err != nil {
					return nil, err
				}
				newObject, err := db.Storage.Objects.decode([]byte(data))
				if err == nil {
					res = append(res, newObject)
				}
			}
		}

		return db.Storage.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Peek  :
func (db *RedisStorage) Peek(key string, now int64) (int64, int64) {
	previous, err := db.redisdb.Get(key).Result()
	if err != nil || err == redis.Nil {
		return now, 0
	}

	oldObject, err := db.Storage.Objects.decode([]byte(previous))
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set  :
func (db *RedisStorage) Set(key string, index string, now int64, data string) (string, error) {
	created, updated := db.Peek(key, now)
	err := db.redisdb.Set(
		key,
		db.Storage.Objects.new(&Object{
			Created: created,
			Updated: updated,
			Index:   index,
			Data:    data,
		}), 0).Err()
	if err != nil {
		return "", err
	}

	return index, nil
}

// Del  :
func (db *RedisStorage) Del(key string) error {
	_, err := db.redisdb.Get(key).Result()
	if err != nil && err == redis.Nil {
		return errors.New("samo: not found")
	}

	if err != nil {
		return err
	}

	return db.redisdb.Del(key).Err()
}
