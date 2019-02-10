package samo

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// Message expected from websocket connections
type Message struct {
	Op    string `json:"op"`
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
