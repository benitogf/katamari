package samo

import (
	"errors"
	"strings"

	"github.com/gobwas/glob"
)

// Apply filter function
// type for functions will serve as filters
// key: the key to filter
// data: the data received or about to be sent
// returns
// data: to be stored or sent to the client
// error: will prevent data to pass the filter
type apply func(key string, data []byte) ([]byte, error)

// Filter path -> match
type filter struct {
	glob  glob.Glob
	path  string
	apply apply
}

// Router group of filters
type router []filter

// Filters read and write
type filters struct {
	Write router
	Read  router
}

// https://github.com/golang/go/issues/11862

// WriteFilter add a filter that triggers on write
func (app *Server) WriteFilter(path string, apply apply) {
	app.filters.Write = append(app.filters.Write, filter{
		glob:  glob.MustCompile(path),
		path:  path,
		apply: apply,
	})
}

// ReadFilter add a filter that runs before sending a read result
func (app *Server) ReadFilter(path string, apply apply) {
	app.filters.Read = append(app.filters.Read, filter{
		glob:  glob.MustCompile(path),
		path:  path,
		apply: apply,
	})
}

func (r router) check(key string, data []byte, static bool) ([]byte, error) {
	match := -1
	countKey := strings.Count(key, "/")
	for i, filter := range r {
		if filter.glob.Match(key) && countKey == strings.Count(filter.path, "/") {
			match = i
			break
		}
	}

	if match == -1 && !static {
		return data, nil
	}

	if match == -1 && static {
		return nil, errors.New("route not defined, static mode, key:" + key)
	}

	return r[match].apply(key, data)
}
