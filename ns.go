package katamari

import (
	"bufio"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/messages"
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
		client := app.Stream.OpenNs(newClient)
		cache, err := app.Fetch(newClient.Path, newClient.Path)
		if err != nil {
			app.console.Err("katamari: filtered route", err)
			app.Stream.CloseNs(client)
			continue
		}

		// send initial msg
		go app.Stream.WriteNs(client, messages.Encode(cache.Data), true)
		go app.Stream.ReadNs(client)
	}
}
