package samo

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/embed"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
)

// EtcdStorage : composition of storage
type EtcdStorage struct {
	mutex   sync.RWMutex
	Peers   []string
	Path    string
	cli     *clientv3.Client
	server  *embed.Etcd
	timeout time.Duration
	*Storage
}

// Active  :
func (db *EtcdStorage) Active() bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.Storage.Active
}

// Start  :
func (db *EtcdStorage) Start() error {
	var wg sync.WaitGroup
	wg.Add(1)
	db.mutex.Lock()
	defer db.mutex.Unlock()
	var err error
	if db.Storage == nil {
		db.Storage = &Storage{}
	}
	if db.Path == "" {
		db.Path = "data/default.etcd"
	}
	if db.timeout == 0 {
		db.timeout = 30 * time.Second
	}
	if db.Storage.Separator == "" {
		db.Storage.Separator = "/"
	}
	db.Storage.Active = true
	if len(db.Peers) == 0 {
		db.Peers = []string{"localhost:2379"}
	}
	cfg := embed.NewConfig()
	cfg.Dir = db.Path
	cfg.Logger = "zap"
	cfg.Debug = false
	cfg.LogOutputs = []string{db.Path + "/LOG"}
	db.server, err = embed.StartEtcd(cfg)
	if err != nil {
		return err
	}
	select {
	case <-db.server.Server.ReadyNotify():
		db.cli, err = clientv3.New(clientv3.Config{
			Endpoints:   db.Peers,
			DialTimeout: 5 * time.Second,
		})
		wg.Done()
	case <-time.After(5 * time.Second):
		db.server.Server.Stop()
		err = errors.New("etcd embed server took too long to start")
		wg.Done()
	case <-db.server.Err():
		err = errors.New("etcd embed server error")
		wg.Done()
	}
	wg.Wait()

	return err
}

// Close  :
func (db *EtcdStorage) Close() {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.cli.Close()
	db.server.Close()
	db.Storage.Active = false
}

// Clear  :
func (db *EtcdStorage) Clear() {
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	_, err := db.cli.Delete(ctx, "", clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatal(err)
	}
}

// Keys  :
func (db *EtcdStorage) Keys() ([]byte, error) {
	stats := Stats{}
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

	return db.Storage.Objects.encode(stats)
}

// Get :
func (db *EtcdStorage) Get(mode string, key string) ([]byte, error) {
	if mode == "sa" {
		ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
		resp, err := db.cli.Get(ctx, key)
		cancel()
		if err != nil {
			return []byte(""), err
		}
		if len(resp.Kvs) == 0 {
			return []byte(""), errors.New("samo: not found")
		}

		return resp.Kvs[0].Value, nil
	}

	if mode == "mo" {
		res := []Object{}
		ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
		resp, err := db.cli.Get(ctx, key, clientv3.WithPrefix())
		cancel()
		if err != nil {
			return []byte(""), err
		}
		for _, ev := range resp.Kvs {
			if db.Storage.Keys.isSub(key, string(ev.Key), db.Storage.Separator) {
				newObject, err := db.Storage.Objects.decode(ev.Value)
				if err == nil {
					res = append(res, newObject)
				}
			}
		}

		return db.Storage.Objects.encode(res)
	}

	return []byte(""), errors.New("samo: unrecognized mode: " + mode)
}

// Peek will check the object stored in the key if any, returns created and updated times accordingly
func (db *EtcdStorage) Peek(key string, now int64) (int64, int64) {
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	resp, err := db.cli.Get(ctx, key)
	cancel()
	if err != nil || len(resp.Kvs) == 0 {
		return now, 0
	}

	oldObject, err := db.Storage.Objects.decode(resp.Kvs[0].Value)
	if err != nil {
		return now, 0
	}

	return oldObject.Created, now
}

// Set  :
func (db *EtcdStorage) Set(key string, index string, now int64, data string) (string, error) {
	created, updated := db.Peek(key, now)
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	_, err := db.cli.Put(ctx, key, string(db.Storage.Objects.new(&Object{
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
func (db *EtcdStorage) Del(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	_, err := db.cli.Get(ctx, key)
	cancel()
	if err == rpctypes.ErrEmptyKey {
		return errors.New("samo: not found")
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

// Watch :
func (db *EtcdStorage) Watch(key string) interface{} {
	return db.cli.Watch(context.Background(), "", clientv3.WithPrefix())
}
