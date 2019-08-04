package samo

import (
	"errors"
	"strings"

	"github.com/gobwas/glob"
)

// Apply filter function
type Apply func(key string, data []byte) ([]byte, error)

// Filter path -> match
type Filter struct {
	glob  glob.Glob
	path  string
	apply Apply
}

// Router group of filters
type Router []Filter

// Filters read and write
type Filters struct {
	Write Router
	Read  Router
}

// https://github.com/golang/go/issues/11862

// WriteFilter add a filter that triggers on write
func (app *Server) WriteFilter(path string, apply Apply) {
	app.Filters.Write = append(app.Filters.Write, Filter{
		glob:  glob.MustCompile(path),
		path:  path,
		apply: apply,
	})
}

// ReadFilter add a filter that runs before sending a read result
func (app *Server) ReadFilter(path string, apply Apply) {
	app.Filters.Read = append(app.Filters.Read, Filter{
		glob:  glob.MustCompile(path),
		path:  path,
		apply: apply,
	})
}

func (r Router) check(key string, data []byte, static bool) ([]byte, error) {
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
