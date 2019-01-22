package samo

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (app *Server) getKeys() ([]byte, error) {
	iter := app.db.NewIterator(nil, nil)
	stats := Stats{}
	for iter.Next() {
		stats.Keys = append(stats.Keys, string(iter.Key()))
	}
	iter.Release()
	err := iter.Error()
	if err == nil {
		if stats.Keys == nil {
			stats.Keys = []string{}
		}
		respJSON, err := json.Marshal(stats)
		if err == nil {
			return respJSON, nil
		}
	}

	return nil, err
}

func (app *Server) getData(mode string, key string) []byte {
	switch mode {
	case "sa":
		data, err := app.db.Get([]byte(key), nil)
		if err == nil {
			return data
		}
		app.console.err("getError["+mode+"/"+key+"]", err)
	case "mo":
		iter := app.db.NewIterator(util.BytesPrefix([]byte(key+app.separator)), nil)
		res := []Object{}
		for iter.Next() {
			if app.isMO(key, string(iter.Key()), app.separator) {
				var newObject Object
				err := json.Unmarshal(iter.Value(), &newObject)
				if err == nil {
					res = append(res, newObject)
				} else {
					app.console.err("getError["+mode+"/"+key+"]", err)
				}
			}
		}
		iter.Release()
		err := iter.Error()
		if err == nil {
			data, err := json.Marshal(res)
			if err == nil {
				return data
			}
		}
	}

	return []byte("")
}

func (app *Server) setData(mode string, key string, index string, subIndex string, data string) (string, error) {
	now := time.Now().UTC().UnixNano()
	updated := now

	if mode == "sa" {
		index = app.extractMoIndex(key, app.separator)
	}
	if mode == "mo" {
		if index == "" {
			index = strconv.FormatInt(now, 16) + subIndex
		}
		key += app.separator + index
	}

	if !app.checkArchetype(key, data, app.Archetypes) {
		app.console.err("setError["+mode+"/"+key+"]", "improper data")
		return "", errors.New("SAMO: dataArchtypeError improper data")
	}

	previous, err := app.db.Get([]byte(key), nil)
	created := now
	if err != nil && err.Error() == "leveldb: not found" {
		updated = 0
	}

	if err == nil {
		var oldObject Object
		err = json.Unmarshal(previous, &oldObject)
		if err == nil {
			created = oldObject.Created
		}
	}

	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(Object{
		Created: created,
		Updated: updated,
		Index:   index,
		Data:    data,
	})

	err = app.db.Put(
		[]byte(key),
		dataBytes.Bytes(), nil)
	if err == nil {
		app.console.log("set[" + mode + "/" + key + "]")
		return index, nil
	}

	app.console.err("setError["+mode+"/"+key+"]", err)
	return "", err
}

func (app *Server) delData(mode string, key string, index string) error {
	if mode == "mo" {
		key += app.separator + index
	}

	_, err := app.db.Get([]byte(key), nil)
	if err == nil {
		err := app.db.Delete([]byte(key), nil)
		if err == nil {
			app.console.log("delete[" + mode + "/" + key + "]")
			return nil
		}
	}

	app.console.err("deleteError["+mode+"/"+key+"]", err)
	return err
}
