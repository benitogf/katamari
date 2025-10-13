package io

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/benitogf/katamari/client"
	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/objects"
	"github.com/goccy/go-json"
)

type PostBody struct {
	Data string `json:"data"`
}

func RemoteSet[T any](_client *http.Client, ssl bool, host string, path string, item T) error {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	if isList {
		return errors.New("RemoteSet[" + path + "]: path is a list")
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		log.Println("RemoteSet["+path+"]: failed to marshal data", err)
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(jsonData)

	postBody := PostBody{
		Data: encoded,
	}
	jsonPostBodyData, err := json.Marshal(postBody)
	if err != nil {
		log.Println("RemoteSet["+path+"]: failed to marshal data", err)
		return err
	}

	var resp *http.Response
	if ssl {
		resp, err = _client.Post("https://"+host+"/"+path, "application/json", bytes.NewReader(jsonPostBodyData))
	} else {
		resp, err = _client.Post("http://"+host+"/"+path, "application/json", bytes.NewReader(jsonPostBodyData))
	}
	_ = resp
	// if err != nil {
	// 	log.Println("RemoteSet["+path+"]: failed to post to remote", err)
	// 	return err
	// }

	// defer resp.Body.Close()
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Println("RemoteSet["+path+"]: failed to read response", string(body), err)
	// 	return err
	// }
	// log.Println("RemoteSet["+path+"]: sent data", string(body), encoded)
	return err
}

func RemotePush[T any](_client *http.Client, ssl bool, host string, path string, item T) error {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	if !isList {
		return errors.New("RemotePush[" + path + "]: path is not a list")
	}

	_path := key.Build(path)

	jsonData, err := json.Marshal(item)
	if err != nil {
		log.Println("RemotePush["+path+"]: failed to marshal data", err)
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(jsonData)

	postBody := PostBody{
		Data: encoded,
	}
	jsonPostBodyData, err := json.Marshal(postBody)
	if err != nil {
		log.Println("RemotePush["+path+"]: failed to marshal data", err)
		return err
	}

	if ssl {
		_, err = _client.Post("https://"+host+"/"+_path, "application/json", bytes.NewReader(jsonPostBodyData))
	} else {
		_, err = _client.Post("http://"+host+"/"+_path, "application/json", bytes.NewReader(jsonPostBodyData))
	}
	return err
}

func RemoteGet[T any](_client *http.Client, ssl bool, host string, path string) (client.Meta[T], error) {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	if isList {
		return client.Meta[T]{}, errors.New("GetFrom[" + path + "]: path is a list")
	}

	var resp *http.Response
	var err error
	if ssl {
		resp, err = _client.Get("https://" + host + "/" + path)
	} else {
		resp, err = _client.Get("http://" + host + "/" + path)
	}
	if err != nil {
		log.Println("GetFrom["+path+"]: failed to get from remote", err)
		return client.Meta[T]{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("GetFrom["+path+"]: failed to read response", string(body), err)
		return client.Meta[T]{}, err
	}
	obj, err := objects.Decode(body)
	if err != nil {
		log.Println("GetFrom["+path+"]: failed to decode data", string(body), err)
		return client.Meta[T]{}, err
	}
	var item T
	err = json.Unmarshal([]byte(obj.Data), &item)
	if err != nil {
		log.Println("GetFrom["+path+"]: failed to unmarshal data", err)
		return client.Meta[T]{}, err
	}
	return client.Meta[T]{
		Created: obj.Created,
		Updated: obj.Updated,
		Index:   obj.Index,
		Data:    item,
	}, nil
}

func RemoteGetList[T any](_client *http.Client, ssl bool, host string, path string) ([]client.Meta[T], error) {
	lastPath := key.LastIndex(path)
	isList := lastPath == "*"

	if !isList {
		return []client.Meta[T]{}, errors.New("GetListFrom[" + path + "]: path is not a list")
	}

	var resp *http.Response
	var err error
	if ssl {
		resp, err = _client.Get("https://" + host + "/" + path)
	} else {
		resp, err = _client.Get("http://" + host + "/" + path)
	}
	if err != nil {
		log.Println("GetListFrom["+path+"]: failed to get from remote", err)
		return []client.Meta[T]{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("GetListFrom["+path+"]: failed to read response", err)
		return []client.Meta[T]{}, err
	}
	objs, err := objects.DecodeList(body)
	if err != nil {
		log.Println("GetListFrom["+path+"]: failed to decode data", err)
		return []client.Meta[T]{}, err
	}
	result := []client.Meta[T]{}
	for _, obj := range objs {
		var item T
		err = json.Unmarshal([]byte(obj.Data), &item)
		if err != nil {
			log.Println("GetListFrom["+path+"]: failed to unmarshal data", err)
			continue
		}
		result = append(result, client.Meta[T]{
			Created: obj.Created,
			Updated: obj.Updated,
			Index:   obj.Index,
			Data:    item,
		})
	}
	return result, nil
}
