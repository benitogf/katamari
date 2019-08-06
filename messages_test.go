package samo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegex(t *testing.T) {
	require.True(t, keyGlobRegex.MatchString("*"))
	require.True(t, keyGlobRegex.MatchString("*/a"))
	require.True(t, keyGlobRegex.MatchString("a/b/*"))
	require.True(t, keyGlobRegex.MatchString("a/b/c"))
	require.False(t, keyGlobRegex.MatchString("/a/b/c"))
	require.False(t, keyGlobRegex.MatchString("a/b/c/"))
	require.False(t, keyGlobRegex.MatchString("a:b/c"))
}
