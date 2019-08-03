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

func (o *Objects) sort(obj []Object) func(i, j int) bool {
	return func(i, j int) bool {
		created := obj[i].Created > obj[j].Created
		updated := obj[i].Updated > obj[j].Updated
		if updated && obj[j].Updated != 0 {
			return true
		}

		return created
	}
}

func (o *Objects) encode(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(""), err
	}

	return data, nil
}

func (o *Objects) decode(data []byte) (Object, error) {
	var obj Object
	err := json.Unmarshal(data, &obj)

	return obj, err
}

func (o *Objects) new(obj *Object) []byte {
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(obj)

	return dataBytes.Bytes()
}
