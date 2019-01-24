package samo

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegex(t *testing.T) {
	app := Server{}
	separator := "/"
	rr, _ := regexp.Compile("^" + app.helpers.makeRouteRegex(separator) + "$")
	require.True(t, rr.MatchString("a/b/c"))
	require.False(t, rr.MatchString("/a/b/c"))
	require.False(t, rr.MatchString("a/b/c/"))
	require.False(t, rr.MatchString("a:b/c"))
	separator = ":"
	rr, _ = regexp.Compile("^" + app.helpers.makeRouteRegex(separator) + "$")
	require.True(t, rr.MatchString("a:b:c"))
	require.False(t, rr.MatchString("a:b/c"))
}

func TestValidKey(t *testing.T) {
	app := Server{}
	require.True(t, app.helpers.validKey("test", "/"))
	require.True(t, app.helpers.validKey("test/1", "/"))
	require.False(t, app.helpers.validKey("test//1", "/"))
	require.False(t, app.helpers.validKey("test///1", "/"))
}

func TestIsMo(t *testing.T) {
	app := Server{}
	require.True(t, app.helpers.IsMO("thing", "thing/123", "/"))
	require.True(t, app.helpers.IsMO("thing/123", "thing/123/123", "/"))
	require.False(t, app.helpers.IsMO("thing/123", "thing/12", "/"))
	require.False(t, app.helpers.IsMO("thing/1", "thing/123", "/"))
	require.False(t, app.helpers.IsMO("thing/123", "thing/123/123/123", "/"))
}
