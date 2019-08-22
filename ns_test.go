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
	app.forcePatch = true
	app.NamedSocket = "samotest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	client, err := nsocket.Dial("samotest", "test/**")
	require.NoError(t, err)
	_, err = client.Read()
	require.Error(t, err)
}

func TestStreamFilteredNsKey(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.forcePatch = true
	app.Static = true
	app.NamedSocket = "samotest"
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	client, err := nsocket.Dial("samotest", "test/*")
	require.NoError(t, err)
	_, err = client.Read()
	require.Error(t, err)
}

func TestStreamNoNss(t *testing.T) {
	app := Server{}
	app.Silence = true
	app.forcePatch = true
	app.Static = true
	app.Start("localhost:9889")
	defer app.Close(os.Interrupt)
	_, err := nsocket.Dial("samotest", "test/*")
	require.Error(t, err)
}
