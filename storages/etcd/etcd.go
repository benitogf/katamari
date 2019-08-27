package etcd

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/key"
	"go.etcd.io/etcd/clientv3"

	// "go.etcd.io/etcd/embed"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
)

// Etcd : composition of storage
type Etcd struct {
	mutex sync.RWMutex
	Peers []string
	Path  string
	cli   *clientv3.Client
	// server  *embed.Etcd
	timeout time.Duration
	watcher katamari.StorageChan
	// OnlyClient bool
	Debug bool
	*katamari.Storage
}

// Active  :
func (db *Etcd) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *Etcd) Start() error {
	var wg sync.WaitGroup
	db.mutex.Lock()
	defer db.mutex.Unlock()
	var err error
	if db.Storage == nil {
		db.Storage = &katamari.Storage{}
	}
	if db.watcher == nil {
		db.watcher = make(katamari.StorageChan)
	}
	if db.Path == "" {
		db.Path = "data/default.etcd"
	}
	if db.timeout == 0 {
		db.timeout = 30 * time.Second
	}
	if len(db.Peers) == 0 {
		db.Peers = []string{"localhost:2379"}
	}
	// if !db.OnlyClient {
	// wg.Add(1)
	// cfg := embed.NewConfig()
	// cfg.Dir = db.Path
	// cfg.Logger = "zap"
	// cfg.Debug = false
	// cfg.LogLevel = "fatal"
	// // cfg.LogOutputs = []string{db.Path + "/LOG"}
	// db.server, err = embed.StartEtcd(cfg)
	// if err != nil {
	// 	return err
	// }
	// select {
	// case <-db.server.Server.ReadyNotify():
	// 	wg.Done()
	// case <-time.After(5 * time.Second):
	// 	db.server.Server.Stop()
	// 	err = errors.New("etcd embed server took too long to start")
	// 	wg.Done()
	// case <-db.server.Err():
	// 	err = errors.New("etcd embed server error")
	// 	wg.Done()
	// }
	// }
	wg.Wait()
	db.cli, err = clientv3.New(clientv3.Config{
		Endpoints:   db.Peers,
		DialTimeout: 5 * time.Second,
	})
	go func() {
		for wresp := range db.cli.Watch(context.Background(), "", clientv3.WithPrefix()) {
			for _, ev := range wresp.Events {
				db.watcher <- katamari.StorageEvent{Key: string(ev.Kv.Key), Operation: string(ev.Type)}
			}
			if !db.Active() {
				return
			}
		}
	}()
	db.Storage.Active = true

	return err
}

// Close  :
func (db *Etcd) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.cli.Close()
	// if !db.OnlyClient {
	// db.server.Close()
	// }
	close(db.watcher)
	db.Storage.Active = false
}

// Clear  :
func (db *Etcd) Clear() {
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	_, err := db.cli.Delete(ctx, "", clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatal(err)
	}
}

// Keys  :
func (db *Etcd) Keys() ([]byte, error) {
	stats := katamari.Stats{}
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	resp, err := db.cli.Get(ctx, "", clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}
	for _, ev := range resp.Kvs {
		stats.Keys = append(stats.Keys, string(ev.Key))
	}

	if stats.Keys == nil {
		stats.Keys = []string{}
	}
	sort.Slice(stats.Keys, func(i, j int) bool {
		return strings.ToLower(stats.Keys[i]) < strings.ToLower(stats.Keys[j])
	})

	return db.Storage.Objects.Encode(stats)
}

// Get :
func (db *Etcd) Get(path string) ([]byte, error) {
	if !strings.Contains(path, "*") {
		ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
		resp, err := db.cli.Get(ctx, path)
		cancel()
		if err != nil {
			return []byte(""), err
		}
		if len(resp.Kvs) == 0 {
			return []byte(""), errors.New("katamari: not found")
		}

		return resp.Kvs[0].Value, nil
	}

	res := []katamari.Object{}
	globPrefixKey := strings.Split(path, "*")[0]
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	resp, err := db.cli.Get(ctx, globPrefixKey, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return []byte(""), err
	}
	for _, ev := range resp.Kvs {
		if key.Match(path, string(ev.Key)) {
			newObject, err := db.Storage.Objects.Decode(ev.Value)
			if err == nil {
				res = append(res, newObject)
			}
		}
	}

	sort.Slice(res, db.Storage.Objects.Sort(res))

	return db.Storage.Objects.Encode(res)
}

// Peek will check the object stored in the key if any, returns created and updated times accordingly
func (db *Etcd) Peek(key string, now int64) (int64, int64) {
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	resp, err := db.cli.Get(ctx, key)
	cancel()
	if err != nil || len(resp.Kvs) == 0 {
		return now, 0
	}

	oldObject, err := db.Storage.Objects.Decode(resp.Kvs[0].Value)
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set  :
func (db *Etcd) Set(path string, data string) (string, error) {
	now := time.Now().UTC().UnixNano()
	index := key.LastIndex(path)
	created, updated := db.Peek(path, now)
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	_, err := db.cli.Put(ctx, path, string(db.Storage.Objects.New(&katamari.Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	})))
	cancel()
	if err != nil {
		return "", err
	}
	return index, nil
}

// Del  :
func (db *Etcd) Del(key string) error {
	if !strings.Contains(key, "*") {
		ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
		_, err := db.cli.Get(ctx, key)
		cancel()
		if err == rpctypes.ErrEmptyKey {
			return errors.New("katamari: not found")
		}
		if err != nil {
			return err
		}

		ctx, cancel = context.WithTimeout(context.Background(), db.timeout)
		_, err = db.cli.Delete(ctx, key)
		cancel()
		if err != nil {
			return err
		}

		return nil
	}

	globPrefixKey := strings.Split(key, "*")[0]
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	_, err := db.cli.Delete(ctx, globPrefixKey, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return err
	}

	return nil
}

// Watch :
func (db *Etcd) Watch() katamari.StorageChan {
	return db.watcher
}
