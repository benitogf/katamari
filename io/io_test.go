package io_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/io"
	"github.com/stretchr/testify/require"
)

type Thing struct {
	This string `json:"this"`
	That string `json:"that"`
}

const THING1_PATH = "thing1"
const THING2_PATH = "thing2"
const THINGS_BASE_PATH = "things"
const THINGS_PATH = THINGS_BASE_PATH + "/*"

func TestIObasic(t *testing.T) {
	server := &katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)
	err := io.Set(server, THING1_PATH, Thing{
		This: "this",
		That: "that",
	})
	require.NoError(t, err)
	err = io.Set(server, THING2_PATH, Thing{
		This: "here",
		That: "there",
	})
	require.NoError(t, err)

	thing1, err := io.Get[Thing](server, THING1_PATH)
	require.NoError(t, err)

	require.Equal(t, "this", thing1.Data.This)
	require.Equal(t, "that", thing1.Data.That)

	thing2, err := io.Get[Thing](server, THING2_PATH)
	require.NoError(t, err)

	require.Equal(t, "here", thing2.Data.This)
	require.Equal(t, "there", thing2.Data.That)

	err = io.Push(server, THINGS_PATH, thing1.Data)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		time.Sleep(10 * time.Millisecond)
	}
	err = io.Push(server, THINGS_PATH, thing2.Data)
	require.NoError(t, err)

	things, err := io.GetList[Thing](server, THINGS_PATH)
	require.NoError(t, err)
	require.Equal(t, 2, len(things))
	require.Equal(t, "here", things[0].Data.This)
	require.Equal(t, "this", things[1].Data.This)

	err = io.Set(server, string(THINGS_BASE_PATH)+"/what", Thing{
		This: "what",
		That: "how",
	})
	require.NoError(t, err)

	things, err = io.GetList[Thing](server, THINGS_PATH)
	require.NoError(t, err)
	require.Equal(t, 3, len(things))
	require.Equal(t, "what", things[0].Data.This)
	require.Equal(t, "this", things[2].Data.This)
}

func TestRemoteIO(t *testing.T) {
	server := &katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)

	err := io.RemoteSet(server.Client, false, server.Address, THING1_PATH, Thing{
		This: "this",
		That: "that",
	})
	require.NoError(t, err)
	err = io.RemoteSet(server.Client, false, server.Address, THING2_PATH, Thing{
		This: "here",
		That: "there",
	})
	require.NoError(t, err)

	thing1, err := io.RemoteGet[Thing](server.Client, false, server.Address, THING1_PATH)
	require.NoError(t, err)

	require.Equal(t, "this", thing1.Data.This)
	require.Equal(t, "that", thing1.Data.That)

	thing2, err := io.RemoteGet[Thing](server.Client, false, server.Address, THING2_PATH)
	require.NoError(t, err)

	require.Equal(t, "here", thing2.Data.This)
	require.Equal(t, "there", thing2.Data.That)

	err = io.RemotePush(server.Client, false, server.Address, THINGS_PATH, thing1.Data)
	require.NoError(t, err)
	err = io.RemotePush(server.Client, false, server.Address, THINGS_PATH, thing2.Data)
	require.NoError(t, err)

	things, err := io.RemoteGetList[Thing](server.Client, false, server.Address, THINGS_PATH)
	require.NoError(t, err)
	require.Equal(t, 2, len(things))

	err = io.RemoteSet(server.Client, false, server.Address, string(THINGS_BASE_PATH)+"/what", Thing{
		This: "what",
		That: "how",
	})
	require.NoError(t, err)

	things, err = io.RemoteGetList[Thing](server.Client, false, server.Address, THINGS_PATH)
	require.NoError(t, err)
	require.Equal(t, 3, len(things))
	require.Equal(t, "what", things[0].Data.This)
	require.Equal(t, "this", things[2].Data.This)
}
