package samo

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegex(t *testing.T) {
	app := Server{}
	app.separator = "/"
	rr, _ := regexp.Compile("^" + app.makeRouteRegex() + "$")
	require.True(t, rr.MatchString("a/b/c"))
	require.False(t, rr.MatchString("/a/b/c"))
	require.False(t, rr.MatchString("a/b/c/"))
	require.False(t, rr.MatchString("a:b/c"))
	app.separator = ":"
	rr, _ = regexp.Compile("^" + app.makeRouteRegex() + "$")
	require.True(t, rr.MatchString("a:b:c"))
	require.False(t, rr.MatchString("a:b/c"))
}
