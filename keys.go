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

var keyGlobRegex = regexp.MustCompile(`^[a-zA-Z\*\d]$|^[a-zA-Z\*\d][a-zA-Z\*\d\/]+[a-zA-Z\*\d]$`)
var globsCache = map[string]glob.Glob{}

// IsValid checks that the key pattern issuported
func (keys *Keys) IsValid(key string) bool {
	if strings.Contains(key, "//") || strings.Contains(key, "**") {
		return false
	}

	return keyGlobRegex.MatchString(key)
}

// Match checks that the key is part of the path
func (keys *Keys) Match(path string, key string) bool {
	if !strings.Contains(path, "*") {
		return false
	}
	globPath, found := globsCache[path]
	if !found {
		globPath = glob.MustCompile(path)
		globsCache[path] = globPath
	}
	countPath := strings.Count(path, "/")
	countKey := strings.Count(key, "/")
	return globPath.Match(key) && countPath == countKey
}

// lastIndex will return the last sub path of the key
func (keys *Keys) lastIndex(key string) string {
	return key[strings.LastIndexAny(key, "/")+1:]
}

// Build checks the key glob pattern
func (keys *Keys) Build(key string) string {
	now := time.Now().UTC().UnixNano()
	if !strings.Contains(key, "*") {
		return key
	}

	index := strconv.FormatInt(now, 16)
	return strings.Replace(key, "/*", "/"+index, 1)
}
