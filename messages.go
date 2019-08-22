package samo

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// Message sent through websocket connections
type message struct {
	Data    string `json:"data"`
	Version string `json:"version"`
}

// Messages encoding and decoding
type messages struct{}

// write base64 string from bytes
func (messages *messages) encode(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func (messages *messages) decodeTest(data []byte) (message, error) {
	var wsEvent message
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

func (messages *messages) decode(r io.Reader) (message, error) {
	var httpEvent message
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
