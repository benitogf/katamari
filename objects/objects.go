package objects

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
)

// Object : data structure of elements
type Object struct {
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	Index   string `json:"index"`
	Data    string `json:"data"`
}

// EmptyObject byte array value
var EmptyObject = []byte(`{ "created": 0, "updated": 0, "index": "", "data": "e30=" }`)

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
func Decode(data []byte) (Object, error) {
	var obj Object
	err := json.Unmarshal(data, &obj)

	return obj, err
}

// DecodeList json objects
func DecodeList(data []byte) ([]Object, error) {
	var objects []Object
	err := json.Unmarshal(data, &objects)
	if err != nil {
		return objects, err
	}
	for i := range objects {
		aux, err := base64.StdEncoding.DecodeString(objects[i].Data)
		if err != nil {
			break
		}
		objects[i].Data = string(aux)
	}
	return objects, err
}

// New object as json
func New(obj *Object) []byte {
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(obj)

	return dataBytes.Bytes()
}
