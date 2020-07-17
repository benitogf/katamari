package key

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegex(t *testing.T) {
	require.True(t, GlobRegex.MatchString("*"))
	require.True(t, GlobRegex.MatchString("*/a"))
	require.True(t, GlobRegex.MatchString("a/b/*"))
	require.True(t, GlobRegex.MatchString("a/b/c"))
	require.False(t, GlobRegex.MatchString("/a/b/c"))
	require.False(t, GlobRegex.MatchString("a/b/c/"))
	require.False(t, GlobRegex.MatchString("a:b/c"))
}

func TestKeyIsValid(t *testing.T) {
	require.True(t, IsValid("test"))
	require.True(t, IsValid("test/1"))
	require.False(t, IsValid("test//1"))
	require.False(t, IsValid("test///1"))
}

func TestKeyMatch(t *testing.T) {
	require.True(t, Match("*", "thing"))
	require.True(t, Match("games/*", "games/*"))
	require.True(t, Match("thing/*", "thing/123"))
	require.True(t, Match("thing/123/*", "thing/123/234"))
	require.True(t, Match("thing/glob/*/*", "thing/glob/test/234"))
	require.True(t, Match("thing/123/*", "thing/123/123"))
	require.False(t, Match("thing/*/*", "thing/123/234/234"))
	require.False(t, Match("thing/123", "thing/12"))
	require.False(t, Match("thing/1", "thing/123"))
	require.False(t, Match("thing/123/*", "thing/123/123/123"))
}
