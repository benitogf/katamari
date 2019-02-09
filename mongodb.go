package samo

import (
	"errors"
	"fmt"
	"sync"
)

type mongodbStorage struct {
	mutex sync.RWMutex
	*Storage
}

// Active  :
func (db *mongodbStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *mongodbStorage) Start() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	db.Storage.Active = true
	return nil
}

// Close  :
func (db *mongodbStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.Storage.Active = false
}

// Clear :
func (db *mongodbStorage) Clear() {

}

// Keys  :
func (db *mongodbStorage) Keys() ([]byte, error) {
	fmt.Println("keys")
	return []byte(""), errors.New("not implemented")
}

// Get :
func (db *mongodbStorage) Get(mode string, key string) ([]byte, error) {
	fmt.Println("get", mode, key)
	return []byte(""), errors.New("not implemented")
}

// Peek  :
func (db *mongodbStorage) Peek(key string, now int64) (int64, int64) {
	fmt.Println("peek", key, now)
	return 0, 0
}

// Set  :
func (db *mongodbStorage) Set(key string, index string, now int64, data string) (string, error) {
	fmt.Println("set", key, index, now, data)
	return "", errors.New("not implemented")
}

// Del  :
func (db *mongodbStorage) Del(key string) error {
	fmt.Println("del", key)
	return errors.New("not implemented")
}
