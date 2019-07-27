package samo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyIsValid(t *testing.T) {
	keys := &Keys{}
	require.True(t, keys.isValid("test", "/"))
	require.True(t, keys.isValid("test/1", "/"))
	require.False(t, keys.isValid("test//1", "/"))
	require.False(t, keys.isValid("test///1", "/"))
}

func TestKeyIsSub(t *testing.T) {
	keys := &Keys{}
	require.True(t, keys.isSub("thing", "thing/123", "/"))
	require.True(t, keys.isSub("thing/*", "thing/123/234", "/"))
	require.True(t, keys.isSub("thing/*/*", "thing/glob/test/234", "/"))
	require.False(t, keys.isSub("thing/*", "thing/123/234/234", "/"))
	require.True(t, keys.isSub("thing/123", "thing/123/123", "/"))
	require.False(t, keys.isSub("thing/123", "thing/12", "/"))
	require.False(t, keys.isSub("thing/1", "thing/123", "/"))
	require.False(t, keys.isSub("thing/123", "thing/123/123/123", "/"))
}
