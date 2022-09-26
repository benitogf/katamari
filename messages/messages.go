package messages

import (
	"errors"
	"io"

	"github.com/goccy/go-json"
)

// Message sent through websocket connections
type Message struct {
	Data     json.RawMessage `json:"data"`
	Version  string          `json:"version"`
	Snapshot bool            `json:"snapshot"`
}

// DecodeTest data (testing function)
func DecodeTest(data []byte) (Message, error) {
	var wsEvent Message
	err := json.Unmarshal(data, &wsEvent)
	if len(wsEvent.Data) == 0 {
		return wsEvent, errors.New("katamari: decode error, empty data")
	}
	if err != nil {
		return wsEvent, err
	}

	return wsEvent, nil
}

// Decode message
func Decode(r io.Reader) (json.RawMessage, error) {
	var httpEvent json.RawMessage
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&httpEvent)
	if err != nil {
		return httpEvent, err
	}
	if len(httpEvent) == 0 {
		return httpEvent, errors.New("katamari: decode error, empty data")
	}

	return httpEvent, nil
}
