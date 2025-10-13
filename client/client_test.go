package client_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/client"
	"github.com/benitogf/katamari/key"
	"github.com/pkg/expect"
	"github.com/stretchr/testify/require"
)

type Device struct {
	Name string `json:"name"`
}

func createDevice(t *testing.T, server *katamari.Server, name string) {
	newKey := key.Build("devices/*")
	newDevice := Device{
		Name: name,
	}

	newDeviceData, err := json.Marshal(newDevice)
	require.NoError(t, err)

	newDeviceDataEncoded := base64.StdEncoding.EncodeToString(newDeviceData)
	server.Storage.Set(newKey, newDeviceDataEncoded)
}

// Sort by created/updated
func SortDevices(obj []client.Meta[Device]) func(i, j int) bool {
	return func(i, j int) bool {
		return obj[i].Created > obj[j].Created
	}
}

func TestClientList(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)
	ctx := t.Context()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go client.Subscribe(ctx, "ws", server.Address, "devices/*",
		func(devices []client.Meta[Device]) {
			sort.Slice(devices, SortDevices(devices))
			if len(devices) > 0 {
				require.Equal(t, "device "+strconv.Itoa(len(devices)-1), devices[0].Data.Name)
			}
			wg.Done()
		})

	wg.Wait()

	for i := range 5 {
		wg.Add(1)
		createDevice(t, &server, "device "+strconv.Itoa(i))
		if runtime.GOOS == "windows" {
			time.Sleep(10 * time.Millisecond)
		}
		wg.Wait()
	}
}

func TestClientClose(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)
	ctx, cancel := context.WithCancel(t.Context())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go client.Subscribe(ctx, "ws", server.Address, "devices/*",
		func(devices []client.Meta[Device]) {
			wg.Done()
		})
	wg.Wait()

	cancel()
	time.Sleep(10 * time.Millisecond) // wait for the connection to be closed
	createDevice(t, &server, "device null")
	time.Sleep(100 * time.Millisecond) // wait to verify that the update is not received
}

func TestClientCloseWhileReconnecting(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)
	ctx, cancel := context.WithCancel(t.Context())

	wg := sync.WaitGroup{}
	wg.Add(1)
	// client.DEBUG = false
	go client.Subscribe(ctx, "ws", server.Address, "devices/*",
		func(devices []client.Meta[Device]) {
			wg.Done()
		})
	wg.Wait()

	// close server and wait for the client to start reconnection attempts
	server.Close(os.Interrupt)
	time.Sleep(1 * time.Second)

	// cancel while reconnecting in progress
	cancel()
	createDevice(t, &server, "device null")
	time.Sleep(1 * time.Second)
}

func TestClientCloseWithoutConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	client.HandshakeTimeout = 10 * time.Millisecond
	go client.Subscribe(ctx, "ws", "notAnIP", "devices/*",
		func(devices []client.Meta[Device]) {
			expect.True(false)
		})

	// wait for retries to stablish connection
	time.Sleep(200 * time.Millisecond)

	// cancel while trying to connect
	cancel()
	time.Sleep(200 * time.Millisecond)
}

func TestClientListCallbackCurry(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	wg := sync.WaitGroup{}

	messagesCount := 0
	devicesCallbacker := func() client.OnMessageCallback[Device] {
		return func(devices []client.Meta[Device]) {
			messagesCount++
			if len(devices) > 0 {
				require.Equal(t, "device "+strconv.Itoa(len(devices)-1), devices[0].Data.Name)
			}
			wg.Done()
		}
	}

	wg.Add(1)
	go client.Subscribe(ctx, "ws", server.Address, "devices/*", devicesCallbacker())

	wg.Wait()

	const NUM_DEVICES = 5
	for i := range NUM_DEVICES {
		wg.Add(1)
		createDevice(t, &server, "device "+strconv.Itoa(i))
		if runtime.GOOS == "windows" {
			time.Sleep(10 * time.Millisecond)
		}
		wg.Wait()
	}

	require.Equal(t, NUM_DEVICES+1, messagesCount)
}
