package samo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/benitogf/mongo-go-driver/mongo/options"

	"github.com/benitogf/mongo-go-driver/bson"
	"github.com/benitogf/mongo-go-driver/mongo"
)

// MongodbStorage : composition of storage
type MongodbStorage struct {
	mutex   sync.RWMutex
	Address string
	mongodb *mongo.Client
	*Storage
}

type document struct {
	key    string
	object []byte
}

func (db *MongodbStorage) trimQuotes(s string) string {
	if len(s) > 0 && s[0] == '"' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	return s
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
	if db.Storage == nil {
		db.Storage = &Storage{}
	}
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	if db.Address == "" {
		db.Address = "localhost:27017"
	}
	db.mongodb, err = mongo.Connect(context.TODO(), "mongodb://"+db.Address)
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
	db.mongodb.Database("samo").Collection("store").Drop(context.TODO())
}

// Keys  :
func (db *MongodbStorage) Keys() ([]byte, error) {
	stats := Stats{}
	// Passing nil as the filter matches all documents in the collection
	cur, err := db.mongodb.Database("samo").Collection("store").Find(context.TODO(), bson.D{})
	if err == mongo.ErrNilDocument {
		return db.Storage.Objects.encode(stats)
	}
	if err != nil {
		return nil, err
	}

	// Finding multiple documents returns a cursor
	// Iterating through the cursor allows us to decode documents one at a time
	for cur.Next(context.TODO()) {
		// create a value into which the single document can be decoded
		stats.Keys = append(stats.Keys, fmt.Sprintf(`%s`, db.trimQuotes(cur.Current.Lookup("key").String())))
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	// Close the cursor once finished
	cur.Close(context.TODO())

	if stats.Keys == nil {
		stats.Keys = []string{}
	}
	sort.Slice(stats.Keys, func(i, j int) bool {
		return strings.ToLower(stats.Keys[i]) < strings.ToLower(stats.Keys[j])
	})

	return db.Storage.Objects.encode(stats)
}

// Get :
func (db *MongodbStorage) Get(mode string, key string) ([]byte, error) {
	if mode == "sa" {
		filter := bson.M{"key": key}
		result := db.mongodb.Database("samo").Collection("store").FindOne(context.TODO(), filter)
		cur, err := result.DecodeBytes()
		if err == mongo.ErrNoDocuments {
			return []byte(""), errors.New("samo: not found")
		}
		if err != nil {
			return []byte(""), err
		}
		_, rawBin := cur.Lookup("object").Binary()
		doc := document{
			key:    cur.Lookup("key").String(),
			object: []byte(rawBin),
		}
		return doc.object, nil
	}

	if mode == "mo" {
		res := []Object{}
		cur, err := db.mongodb.Database("samo").Collection("store").Find(context.TODO(), bson.D{})
		if err == mongo.ErrNilDocument {
			return db.Storage.Objects.encode(res)
		}
		if err != nil {
			return []byte(""), err
		}
		for cur.Next(context.TODO()) {
			_, rawBin := cur.Current.Lookup("object").Binary()
			doc := document{
				key:    cur.Current.Lookup("key").String(),
				object: []byte(rawBin),
			}
			if db.Storage.Keys.isSub(key, doc.key, db.Storage.Separator) {
				newObject, err := db.Storage.Objects.decode(doc.object)
				if err == nil {
					res = append(res, newObject)
				}
			}
		}

		err = cur.Err()
		if err != nil {
			return []byte(""), err
		}

		// Close the cursor once finished
		cur.Close(context.TODO())

		return db.Storage.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Peek  :
func (db *MongodbStorage) Peek(key string, now int64) (int64, int64) {
	filter := bson.M{"key": key}
	result := db.mongodb.Database("samo").Collection("store").FindOne(context.TODO(), filter)
	cur, err := result.DecodeBytes()
	if err != nil {
		return now, 0
	}
	_, rawBin := cur.Lookup("object").Binary()
	previous := document{
		key:    cur.Lookup("key").String(),
		object: []byte(rawBin),
	}
	if err != nil {
		return now, 0
	}

	oldObject, err := db.Storage.Objects.decode(previous.object)
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set  :
func (db *MongodbStorage) Set(key string, index string, now int64, data string) (string, error) {
	created, updated := db.Peek(key, now)
	upsert := true
	_, err := db.mongodb.Database("samo").Collection("store").UpdateOne(context.TODO(),
		bson.M{"key": key},
		bson.M{
			"$set": bson.M{
				"key": key,
				"object": db.Storage.Objects.new(&Object{
					Created: created,
					Updated: updated,
					Index:   index,
					Data:    data,
				}),
			},
		},
		&options.UpdateOptions{
			Upsert: &upsert,
		})

	if err != nil {
		return "", err
	}

	return index, nil
}

// Del  :
func (db *MongodbStorage) Del(key string) error {
	var previous document
	// github.com/benitogf/mongo-go-driver/bson/primitive.E composite literal uses unkeyed fields
	filter := bson.M{"key": key}
	err := db.mongodb.Database("samo").Collection("store").FindOne(context.TODO(), filter).Decode(&previous)
	if err == mongo.ErrNoDocuments {
		return errors.New("samo: not found")
	}

	if err != nil {
		return err
	}
	_, err = db.mongodb.Database("samo").Collection("store").DeleteOne(context.TODO(), filter)
	return err
}

// Watch :
func (db *MongodbStorage) Watch(key string) interface{} {
	return nil
}
