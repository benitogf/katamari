package samo

import (
	"bytes"
	"encoding/json"
)

// Objects provide methods to read from bytes and write to bytes
type Objects struct {
	*Keys
}

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
