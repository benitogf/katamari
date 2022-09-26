package objects

import (
	"bytes"
	"io"

	"github.com/goccy/go-json"
)

// Object : data structure of elements
type Object struct {
	Created int64           `json:"created"`
	Updated int64           `json:"updated"`
	Index   string          `json:"index"`
	Data    json.RawMessage `json:"data"`
}

// EmptyObject byte array value
var EmptyObject = []byte(`{ "created": 0, "updated": 0, "index": "", "data": {} }`)

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Sort by created/updated
func Sort(obj []Object) func(i, j int) bool {
	return func(i, j int) bool {
		maxi := max(obj[i].Updated, obj[i].Created)
		maxj := max(obj[j].Updated, obj[j].Created)
		return maxi > maxj
	}
}

// Encode objects in json
func Encode(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(""), err
	}

	return data, nil
}

// Decode json object
func DecodeRaw(data []byte) (Object, error) {
	var obj Object
	err := json.Unmarshal(data, &obj)

	return obj, err
}

// Decode json object
func Decode(data []byte) (Object, error) {
	var obj Object
	err := json.Unmarshal(data, &obj)
	if err != nil {
		return obj, err
	}

	return obj, err
}

// DecodeFromReader object from io reader
func DecodeFromReader(r io.Reader) (Object, error) {
	var obj Object
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&obj)

	return obj, err
}

// DecodeList json objects
func DecodeList(data []byte) ([]Object, error) {
	var objects []Object
	err := json.Unmarshal(data, &objects)
	if err != nil {
		return objects, err
	}

	return objects, nil
}

// DecodeListFromReader objects from io reader
func DecodeListFromReader(r io.Reader) ([]Object, error) {
	var objs []Object
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&objs)

	return objs, err
}

// DecodeListRaw json objects
func DecodeListRaw(data []byte) ([]Object, error) {
	var objects []Object
	err := json.Unmarshal(data, &objects)
	if err != nil {
		return objects, err
	}

	return objects, nil
}

// New object as json
func New(obj *Object) []byte {
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(obj)

	return dataBytes.Bytes()
}
