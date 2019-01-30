package samo

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"gopkg.in/godo.v2/glob"
)

// Helpers : mixed stateless functions
type Helpers struct{}

func (helpers *Helpers) validKey(key string, separator string) bool {
	// https://stackoverflow.com/a/26792316/6582356
	return !strings.Contains(key, separator+separator)
}

func (helpers *Helpers) extractMoIndex(index string, separator string) string {
	return index[strings.LastIndexAny(index, separator)+1:]
}

// IsMO : checks if the index is part of the key MO
func (helpers *Helpers) IsMO(key string, index string, separator string) bool {
	moIndex := strings.Split(strings.Replace(index, key+separator, "", 1), separator)
	return len(moIndex) == 1 && moIndex[0] != key
}

func (helpers *Helpers) extractNonNil(event map[string]interface{}, field string) string {
	data := ""
	if event[field] != nil {
		data = event[field].(string)
	}

	return data
}

func (helpers *Helpers) makeRouteRegex(separator string) string {
	return "[a-zA-Z\\d][a-zA-Z\\d\\" + separator + "]+[a-zA-Z\\d]"
}

func (helpers *Helpers) accessKey(mode string, key string, index string, separator string) string {
	if mode == "mo" {
		key += separator + index
	}
	return key
}

func (helpers *Helpers) makeKey(mode string, key string, index string, subIndex string, separator string) (string, string, int64) {
	now := time.Now().UTC().UnixNano()

	if mode == "sa" {
		index = helpers.extractMoIndex(key, separator)
		return key, index, now
	}
	if mode == "mo" {
		if index == "" {
			index = strconv.FormatInt(now, 16) + subIndex
		}
		key += separator + index
	}

	return key, index, now
}

func (helpers *Helpers) checkArchetype(key string, index string, data string, static bool, archetypes Archetypes) bool {
	found := ""
	for ar := range archetypes {
		if glob.Globexp(ar).MatchString(key) {
			found = ar
		}
	}
	if found != "" {
		return archetypes[found](index, data)
	}

	return !static
}

func (helpers *Helpers) encodeData(raw []byte) string {
	data := ""
	if len(raw) > 0 {
		data = base64.StdEncoding.EncodeToString(raw)
	}

	return data
}

// Decode : reads base64 encoded ws message
func (helpers *Helpers) Decode(message []byte) (string, error) {
	var wsEvent map[string]interface{}
	err := json.Unmarshal(message, &wsEvent)
	if err != nil {
		return "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(helpers.extractNonNil(wsEvent, "data"))
	if err != nil {
		return "", err
	}
	return strings.Trim(string(decoded), "\n"), nil
}
