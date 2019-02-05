package samo

import (
	"errors"

	"github.com/gobwas/glob"
)

// Apply filter function
type Apply func(key string, data []byte) ([]byte, error)

// Filter path -> match
type Filter struct {
	path  glob.Glob
	apply Apply
}

// Router group of filters
type Router []Filter

// Filters read and write
type Filters struct {
	Receive Router
	Send    Router
}

// https://github.com/golang/go/issues/11862

// ReceiveFilter add a filter that triggers on receive
func (app *Server) ReceiveFilter(path string, apply Apply) {
	app.Filters.Receive = append(app.Filters.Receive, Filter{
		path:  glob.MustCompile(path),
		apply: apply,
	})
}

// SendFilter add a filter that runs before sending
func (app *Server) SendFilter(path string, apply Apply) {
	app.Filters.Send = append(app.Filters.Receive, Filter{
		path:  glob.MustCompile(path),
		apply: apply,
	})
}

func (r Router) check(key string, data []byte, static bool) ([]byte, error) {
	match := -1
	for i, filter := range r {
		if filter.path.Match(key) {
			match = i
			break
		}
	}

	if match == -1 && !static {
		return data, nil
	}

	if match == -1 && static {
		return nil, errors.New("route not defined, static mode")
	}

	return r[match].apply(key, data)
}
