package samo

import (
	"strconv"
	"strings"
	"time"
)

// Keys methods
type Keys struct{}

// isValid checks that the key doesn't contain two consecutive separators
func (keys *Keys) isValid(key string, separator string) bool {
	// https://stackoverflow.com/a/26792316/6582356
	return !strings.Contains(key, separator+separator)
}

// isSub checks if index is a sub path of the key
func (keys *Keys) isSub(key string, index string, separator string) bool {
	moIndex := strings.Split(strings.Replace(index, key+separator, "", 1), separator)
	return len(moIndex) == 1 && moIndex[0] != key
}

// lastIndex will return the last sub path of the key
func (keys *Keys) lastIndex(key string, separator string) string {
	return key[strings.LastIndexAny(key, separator)+1:]
}

// get the key according to the mode
func (keys *Keys) get(mode string, key string, index string, separator string) string {
	if mode == "mo" {
		key += separator + index
	}
	return key
}

// build the key but returns the components as well
func (keys *Keys) build(mode string, key string, index string, subIndex string, separator string) (string, string, int64) {
	now := time.Now().UTC().UnixNano()

	if mode == "sa" {
		index = keys.lastIndex(key, separator)
		return key, index, now
	}
	if mode == "mo" {
		if index == "" {
			// subIndex is used as a way to allow multiple clients adding
			// data to the same key without collisions
			// subIndex = position in the list of clients of the writer
			index = strconv.FormatInt(now, 16) + subIndex
		}
		key += separator + index
	}

	return key, index, now
}
