package messages

import (
	"errors"
	"io"
	"strings"

	"github.com/benitogf/jsonpatch"
	"github.com/benitogf/katamari/objects"
	"github.com/cristalhq/base64"

	"github.com/goccy/go-json"
)

// Message sent through websocket connections
type Message struct {
	Data     string `json:"data"`
	Version  string `json:"version"`
	Snapshot bool   `json:"snapshot"`
}

// Encode to base64 string from bytes
func Encode(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

// Decode message buffer
func DecodeBuffer(data []byte) (Message, error) {
	var message Message
	err := json.Unmarshal(data, &message)
	if err != nil {
		return message, err
	}
	decoded, err := base64.StdEncoding.DecodeString(message.Data)
	if err != nil {
		return message, err
	}
	message.Data = strings.Trim(string(decoded), "\n")

	return message, nil
}

// Decode message reader
func DecodeReader(r io.Reader) (Message, error) {
	var message Message
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&message)
	if err != nil {
		return message, err
	}
	if message.Data == "" {
		return message, errors.New("katamari: empty reader")
	}
	_, err = base64.StdEncoding.DecodeString(message.Data)
	if err != nil {
		return message, err
	}

	return message, nil
}

// Decode a reader into message
// Deprecated: use DecodeReader instead
func Decode(r io.Reader) (Message, error) {
	var message Message
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&message)
	if err != nil {
		return message, err
	}
	if message.Data == "" {
		return message, errors.New("katamari: empty reader")
	}
	_, err = base64.StdEncoding.DecodeString(message.Data)
	if err != nil {
		return message, err
	}

	return message, nil
}

func PatchCache(data []byte, cache string) (string, error) {
	message, err := DecodeBuffer(data)
	if err != nil {
		return cache, err
	}

	if message.Snapshot {
		cache = message.Data
		return cache, nil
	}
	if message.Data == "[]" {
		return cache, nil
	}

	patch, err := jsonpatch.DecodePatch([]byte(message.Data))
	if err != nil || patch == nil {
		return cache, err
	}
	modifiedBytes, err := patch.Apply([]byte(cache))
	if err != nil || modifiedBytes == nil {
		return cache, err
	}

	return string(modifiedBytes), nil
}

func Patch(data []byte, cache string) (string, objects.Object, error) {
	cache, err := PatchCache(data, cache)
	if err != nil {
		return cache, objects.Object{}, err
	}

	result, err := objects.Decode([]byte(cache))
	if err != nil {
		return cache, result, err
	}

	return cache, result, nil
}

func PatchList(data []byte, cache string) (string, []objects.Object, error) {
	cache, err := PatchCache(data, cache)
	if err != nil {
		return cache, []objects.Object{}, err
	}

	result, err := objects.DecodeList([]byte(cache))
	if err != nil {
		return cache, result, err
	}

	return cache, result, nil
}
