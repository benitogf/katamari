package samo

import (
	"os"
	"testing"

	"github.com/benitogf/nsocket"
	"github.com/stretchr/testify/require"
)

func TestStreamInvalidNsKey(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	client, err := nsocket.Dial("samo", "test/**")
	require.NoError(t, err)
	_, err = client.Read()
	require.Error(t, err)
}

func TestStreamFilteredNsKey(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Static = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	client, err := nsocket.Dial("samo", "test/*")
	require.NoError(t, err)
	_, err = client.Read()
	require.Error(t, err)
}
