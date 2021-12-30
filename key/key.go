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
	if path == key {
		return true
	}
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

func Peer(a string, b string) bool {
	return Match(a, b) || Match(b, a)
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

// Decode key to timestamp
func Decode(key string) int64 {
	res, err := strconv.ParseInt(key, 16, 64)
	if err != nil {
		return 0
	}

	return res
}

// Contains find match in an array of paths
func Contains(s []string, e string) bool {
	for _, a := range s {
		if Peer(a, e) {
			return true
		}
	}
	return false
}
