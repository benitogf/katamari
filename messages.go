package katamari

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// Message sent through websocket connections
type Message struct {
	Data    string `json:"data"`
	Version string `json:"version"`
}

// Messages encoding and decoding
type Messages struct{}

// Encode to base64 string from bytes
func (Messages *Messages) Encode(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

// DecodeTest base64 data (testing function)
func (Messages *Messages) DecodeTest(data []byte) (Message, error) {
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
func (Messages *Messages) Decode(r io.Reader) (Message, error) {
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
