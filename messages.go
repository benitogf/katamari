package samo

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// Message expected from websocket connections
type Message struct {
	Op    string `json:"op,omitempty"`
	Index string `json:"index"`
	Data  string `json:"data"`
}

// Messages handle extract, write and read messages
type Messages struct{}

// write base64 string from bytes
func (messages *Messages) encode(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func (messages *Messages) decode(message []byte) (Message, error) {
	var wsEvent Message
	err := json.Unmarshal(message, &wsEvent)
	if err != nil {
		return wsEvent, err
	}
	data, err := base64.StdEncoding.DecodeString(wsEvent.Data)
	if err != nil {
		return wsEvent, err
	}
	wsEvent.Data = strings.Trim(string(data), "\n")

	return wsEvent, nil
}

func (messages *Messages) decodePost(r io.Reader) (Message, error) {
	var httpEvent Message
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&httpEvent)
	if err != nil {
		return httpEvent, err
	}
	if httpEvent.Data == "" {
		return httpEvent, errors.New("samo: empty post data")
	}
	_, err = base64.StdEncoding.DecodeString(httpEvent.Data)
	if err != nil {
		return httpEvent, err
	}

	return httpEvent, nil
}
