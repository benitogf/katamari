package samo

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/glob"
)

// Keys methods
type Keys struct{}

var keyRegex = regexp.MustCompile(`^[a-zA-Z\d]$|^[a-zA-Z\d][a-zA-Z\d\/]+[a-zA-Z\d]$`)
var keyGlobRegex = regexp.MustCompile(`^[a-zA-Z\*\d]$|^[a-zA-Z\*\d][a-zA-Z\*\d\/]+[a-zA-Z\*\d]$`)

// isValid checks that the key doesn't contain two consecutive separators
func (keys *Keys) isValid(glob bool, key string) bool {
	if strings.Contains(key, "//") {
		return false
	}
	if glob {
		return keyGlobRegex.MatchString(key)
	}

	return keyRegex.MatchString(key)
}

// isSub checks if index is a sub path of the key
func (keys *Keys) isSub(key string, index string) bool {
	if strings.Contains(key, "*") {
		re := glob.MustCompile(key)
		keyPath := strings.Split(key, "/")
		indexPath := strings.Split(index, "/")
		return re.Match(index) && len(keyPath) == len(indexPath)
	}
	moIndex := strings.Split(strings.Replace(index, key+"/", "", 1), "/")
	return len(moIndex) == 1 && moIndex[0] != key
}

// lastIndex will return the last sub path of the key
func (keys *Keys) lastIndex(key string) string {
	return key[strings.LastIndexAny(key, "/")+1:]
}

// Build the key but returns the components as well
func (keys *Keys) Build(mode string, key string) string {
	now := time.Now().UTC().UnixNano()
	if mode == "sa" {
		return key
	}

	index := strconv.FormatInt(now, 16)
	key += "/" + index
	return key
}
