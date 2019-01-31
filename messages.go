package samo

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// Messages handle extract, write and read messages
type Messages struct{}

// extract a field value from a message
func (messages *Messages) extract(event map[string]interface{}, field string) string {
	data := ""
	if event[field] != nil {
		data = event[field].(string)
	}

	return data
}

// write base64 string from bytes
func (messages *Messages) write(raw []byte) string {
	data := ""
	if len(raw) > 0 {
		data = base64.StdEncoding.EncodeToString(raw)
	}

	return data
}

// read base64 encoded ws message
func (messages *Messages) read(message []byte) (string, error) {
	var wsEvent map[string]interface{}
	err := json.Unmarshal(message, &wsEvent)
	if err != nil {
		return "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(messages.extract(wsEvent, "data"))
	if err != nil {
		return "", err
	}
	return strings.Trim(string(decoded), "\n"), nil
}
