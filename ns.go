package katamari

import (
	"bufio"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/benitogf/katamari/stream"
	"github.com/benitogf/nsocket"
)

func (app *Server) serveNs() {
	for {
		newConn, err := app.nss.Server.Accept()
		if err != nil {
			app.console.Err(err)
			break
		}
		newClient := &nsocket.Client{
			Conn: newConn,
			Buf:  bufio.NewReadWriter(bufio.NewReader(newConn), bufio.NewWriter(newConn)),
		}
		// handshake message
		newClient.Path, err = newClient.Read()
		if err != nil {
			newClient.Close()
			app.console.Err("failedNsHandshake", err)
			continue
		}
		if !key.IsValid(newClient.Path) {
			newClient.Close()
			app.console.Err("invalidKeyNs[" + newClient.Path + "]")
			continue
		}
		client, poolIndex := app.Stream.OpenNs(newClient)
		// send initial msg
		cache, err := app.Stream.GetPoolCache(newClient.Path)
		if err != nil {
			raw, _ := app.Storage.Get(newClient.Path)
			if len(raw) == 0 {
				raw = objects.EmptyObject
			}
			filteredData, err := app.filters.Read.check(newClient.Path, raw, app.Static)
			if err != nil {
				app.console.Err("katamari: filtered route", err)
				app.Stream.CloseNs(client)
				continue
			}
			newVersion := app.Stream.SetCache(poolIndex, filteredData)
			cache = stream.Cache{
				Version: newVersion,
				Data:    filteredData,
			}
		}

		go app.Stream.WriteNs(client, messages.Encode(cache.Data), true)
		go app.Stream.ReadNs(client)
	}
}
