package samo

import (
	"bytes"
	"encoding/json"
)

// Object : data structure of elements
type Object struct {
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	Index   string `json:"index"`
	Data    string `json:"data"`
}

// Objects provide methods to read from bytes and write to bytes
type Objects struct{}

func (o *Objects) encode(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(""), err
	}

	return data, nil
}

func (o *Objects) read(data []byte) (Object, error) {
	var obj Object
	err := json.Unmarshal(data, &obj)

	return obj, err
}

func (o *Objects) write(obj *Object) []byte {
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(obj)

	return dataBytes.Bytes()
}
