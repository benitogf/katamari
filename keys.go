package samo

import (
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/glob"
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
	if strings.Contains(key, "*") {
		re := glob.MustCompile(key)
		keyPath := strings.Split(key, separator)
		indexPath := strings.Split(index, separator)
		return re.Match(index) && len(keyPath) == len(indexPath)-1
	}
	moIndex := strings.Split(strings.Replace(index, key+separator, "", 1), separator)
	return len(moIndex) == 1 && moIndex[0] != key
}

// lastIndex will return the last sub path of the key
func (keys *Keys) lastIndex(key string, separator string) string {
	return key[strings.LastIndexAny(key, separator)+1:]
}

// Build the key but returns the components as well
func (keys *Keys) Build(mode string, key string, separator string) string {
	now := time.Now().UTC().UnixNano()
	if mode == "sa" {
		return key
	}

	index := strconv.FormatInt(now, 16)
	key += separator + index
	return key
}
