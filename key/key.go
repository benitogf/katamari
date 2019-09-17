package key

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GlobRegex checks for valid glob paths
var GlobRegex = regexp.MustCompile(`^[a-zA-Z\*\d]$|^[a-zA-Z\*\d][a-zA-Z\*\d\/]+[a-zA-Z\*\d]$`)

// IsValid checks that the key pattern issuported
func IsValid(key string) bool {
	if strings.Contains(key, "//") || strings.Contains(key, "**") {
		return false
	}

	return GlobRegex.MatchString(key)
}

// Match checks if a key is part of a path (glob)
func Match(path string, key string) bool {
	if !strings.Contains(path, "*") {
		return false
	}
	match, err := filepath.Match(path, key)
	if err != nil {
		return false
	}
	countPath := strings.Count(path, "/")
	countKey := strings.Count(key, "/")
	return match && countPath == countKey
}

// LastIndex will return the last sub path of the key
func LastIndex(key string) string {
	return key[strings.LastIndexAny(key, "/")+1:]
}

// Build a new key for a path
func Build(key string) string {
	now := time.Now().UTC().UnixNano()
	if !strings.Contains(key, "*") {
		return key
	}

	index := strconv.FormatInt(now, 16)
	return strings.Replace(key, "/*", "/"+index, 1)
}
