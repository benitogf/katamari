package objects

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
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

// DecodeFull json object
func DecodeFull(data []byte) (Object, error) {
	var obj Object
	err := json.Unmarshal(data, &obj)
	aux, err := base64.StdEncoding.DecodeString(obj.Data)
	if err != nil {
		return obj, err
	}

	obj.Data = string(aux)

	return obj, err
}

// DecodeFromReader object from io reader
func DecodeFromReader(r io.Reader) (Object, error) {
	var obj Object
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&obj)

	return obj, err
}

// DecodeListFromReader objects from io reader
func DecodeListFromReader(r io.Reader) ([]Object, error) {
	var objs []Object
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&objs)

	return objs, err
}

// DecodeList json objects
func DecodeList(data []byte) ([]Object, error) {
	var objects []Object
	err := json.Unmarshal(data, &objects)
	if err != nil {
		return objects, err
	}

	return DecodeListData(objects)
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

// DecodeListData will decode the data field in an objects list
func DecodeListData(objects []Object) ([]Object, error) {
	var err error
	for i := range objects {
		var aux []byte
		aux, err = base64.StdEncoding.DecodeString(objects[i].Data)
		if err != nil {
			break
		}
		objects[i].Data = string(aux)
	}

	return objects, err
}

// DecodeRawList json objects
func DecodeRawList(data []byte) ([]Object, error) {
	var objects []Object
	err := json.Unmarshal(data, &objects)
	if err != nil {
		return objects, err
	}
	return objects, err
}

// New object as json
func New(obj *Object) []byte {
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(obj)

	return dataBytes.Bytes()
}
