package io

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/client"
	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
)

func GetList[T any](server *katamari.Server, path string) ([]client.Meta[T], error) {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	var result []client.Meta[T]
	if !isList {
		return result, errors.New("GetList[" + path + "]: path is not a list")
	}
	raw, err := server.Storage.Get(path)
	if err != nil {
		log.Println("GetList["+path+"]: failed to get from storage", err)
		return result, err
	}
	// log.Println("GetList["+path+"]: got from storage", path, string(raw))
	objs, err := objects.DecodeList(raw)
	if err != nil {
		log.Println("GetList["+path+"]: failed to decode data", err)
		return result, err
	}
	for _, obj := range objs {

		var item T
		// log.Println("GetList["+path+"]: unmarshalling data", obj.Created, obj.Data)
		err = json.Unmarshal([]byte(obj.Data), &item)
		if err != nil {
			log.Println("GetList["+path+"]: failed to unmarshal data", err)
			continue
		}
		// log.Println("GetList["+path+"]: marshalled data", item)
		result = append(result, client.Meta[T]{
			Created: obj.Created,
			Updated: obj.Updated,
			Index:   obj.Index,
			Data:    item,
		})
	}
	return result, nil
}

func Get[T any](server *katamari.Server, path string) (client.Meta[T], error) {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	var result client.Meta[T]
	if isList {
		return result, errors.New("Get[" + path + "]: path is a list")
	}

	raw, err := server.Storage.Get(path)
	if err != nil {
		log.Println("Get["+path+"]: failed to get from storage", err)
		return result, err
	}
	// log.Println("Get["+path+"]: got from storage", path, string(raw))
	obj, err := objects.Decode(raw)
	if err != nil {
		log.Println("Get["+path+"]: failed to decode data", err)
		return result, err
	}
	var item T
	err = json.Unmarshal([]byte(obj.Data), &item)
	if err != nil {
		log.Println("Get["+path+"]: failed to unmarshal data", err)
		return result, err
	}
	result = client.Meta[T]{
		Created: obj.Created,
		Updated: obj.Updated,
		Index:   obj.Index,
		Data:    item,
	}
	return result, nil
}

func Set[T any](server *katamari.Server, path string, item T) error {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	if isList {
		return errors.New("Set[" + path + "]: path is a list")
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		log.Println("Set["+path+"]: failed to marshal data", err)
		return err
	}

	// log.Println("Set["+path+"]: marshalled data", string(jsonData))
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	_, err = server.Storage.Set(path, encoded)
	return err
}

func Push[T any](server *katamari.Server, path string, item T) error {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	if !isList {
		return errors.New("Push[" + path + "]: path is not a list")
	}

	_path := key.Build(path)

	jsonData, err := json.Marshal(item)
	if err != nil {
		log.Println("Push["+path+"]: failed to marshal data", err)
		return err
	}
	// log.Println("Push["+path+"]: marshalled data", string(jsonData))
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	_, err = server.Storage.Set(_path, encoded)
	return err
}
