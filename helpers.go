package samo

import (
	"encoding/base64"
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

func (helpers *Helpers) makeIndexes(mode string, key string, index string, subIndex string, separator string) (int64, string, string) {
	now := time.Now().UTC().UnixNano()

	if mode == "sa" {
		index = helpers.extractMoIndex(key, separator)
	}
	if mode == "mo" {
		if index == "" {
			index = strconv.FormatInt(now, 16) + subIndex
		}
		key += separator + index
	}

	return now, key, index
}

func (helpers *Helpers) checkArchetype(key string, data string, archetypes Archetypes) bool {
	found := ""
	for ar := range archetypes {
		if glob.Globexp(ar).MatchString(key) {
			found = ar
		}
	}
	if found != "" {
		return archetypes[found](data)
	}

	return true
}

func (helpers *Helpers) encodeData(raw []byte) string {
	data := ""
	if len(raw) > 0 {
		data = base64.StdEncoding.EncodeToString(raw)
	}

	return data
}
