package katamari

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

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Sort by created/updated
func (o *Objects) Sort(obj []Object) func(i, j int) bool {
	return func(i, j int) bool {
		maxi := max(obj[i].Updated, obj[i].Created)
		maxj := max(obj[j].Updated, obj[j].Created)
		return maxi > maxj
	}
}

// Encode objects in json
func (o *Objects) Encode(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(""), err
	}

	return data, nil
}

// Decode json object
func (o *Objects) Decode(data []byte) (Object, error) {
	var obj Object
	err := json.Unmarshal(data, &obj)

	return obj, err
}

// New object as json
func (o *Objects) New(obj *Object) []byte {
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(obj)

	return dataBytes.Bytes()
}
