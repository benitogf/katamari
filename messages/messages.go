package messages

import (
	"io"
	"log"

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
	if err != nil {
		log.Println("data", string(data), err)
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

	return httpEvent, nil
}
