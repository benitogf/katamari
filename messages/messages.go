package messages

import (
	"errors"
	"io"
	"strings"

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

// DecodeTest base64 data (testing function)
func DecodeTest(data []byte) (Message, error) {
	var wsEvent Message
	err := json.Unmarshal(data, &wsEvent)
	if err != nil {
		return wsEvent, err
	}
	decoded, err := base64.StdEncoding.DecodeString(wsEvent.Data)
	if err != nil {
		return wsEvent, err
	}
	wsEvent.Data = strings.Trim(string(decoded), "\n")

	return wsEvent, nil
}

// Decode message
func Decode(r io.Reader) (Message, error) {
	var httpEvent Message
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&httpEvent)
	if err != nil {
		return httpEvent, err
	}
	if httpEvent.Data == "" {
		return httpEvent, errors.New("katamari: empty post data")
	}
	_, err = base64.StdEncoding.DecodeString(httpEvent.Data)
	if err != nil {
		return httpEvent, err
	}

	return httpEvent, nil
}
