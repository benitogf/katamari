package katamari

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
	app.NamedSocket = "ipctest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	client, err := nsocket.Dial("ipctest", "test/**")
	require.NoError(t, err)
	_, err = client.Read()
	require.Error(t, err)
}

func TestStreamFilteredNsKey(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Static = true
	app.NamedSocket = "ipctest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	client, err := nsocket.Dial("ipctest", "test/*")
	require.NoError(t, err)
	_, err = client.Read()
	require.Error(t, err)
}

func TestStreamNoNss(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.ForcePatch = true
	app.Static = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	_, err := nsocket.Dial("ipctest", "test/*")
	require.Error(t, err)
}
