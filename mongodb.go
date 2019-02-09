package samo

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mongodb/mongo-go-driver/mongo"
)

// MongodbStorage : composition of storage
type MongodbStorage struct {
	mutex   sync.RWMutex
	address string
	mongodb *mongo.Client
	*Storage
}

// Active  :
func (db *MongodbStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *MongodbStorage) Start() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	var err error
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	if db.address == "" {
		db.address = "localhost:27017"
	}
	db.mongodb, err = mongo.NewClient("mongodb://" + db.address)
	if err != nil {
		return err
	}
	err = db.mongodb.Ping(context.TODO(), nil)
	if err != nil {
		return err
	}
	db.Storage.Active = true
	return nil
}

// Close  :
func (db *MongodbStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.mongodb.Disconnect(context.TODO())
	db.Storage.Active = false
}

// Clear :
func (db *MongodbStorage) Clear() {

}

// Keys  :
func (db *MongodbStorage) Keys() ([]byte, error) {
	fmt.Println("keys")
	return []byte(""), errors.New("not implemented")
}

// Get :
func (db *MongodbStorage) Get(mode string, key string) ([]byte, error) {
	fmt.Println("get", mode, key)
	return []byte(""), errors.New("not implemented")
}

// Peek  :
func (db *MongodbStorage) Peek(key string, now int64) (int64, int64) {
	fmt.Println("peek", key, now)
	return 0, 0
}

// Set  :
func (db *MongodbStorage) Set(key string, index string, now int64, data string) (string, error) {
	fmt.Println("set", key, index, now, data)
	return "", errors.New("not implemented")
}

// Del  :
func (db *MongodbStorage) Del(key string) error {
	fmt.Println("del", key)
	return errors.New("not implemented")
}
