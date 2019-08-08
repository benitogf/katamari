package samo

import (
	"bufio"

	"github.com/benitogf/nsocket"
)

func (app *Server) readNs(client *nsocket.Client) {
	for {
		_, err := client.Read()
		if err != nil {
			app.console.Err("readNsError["+client.Path+"]", err)
			break
		}
	}
}

func (app *Server) serveNs() {
	for {
		newConn, err := app.ns.Server.Accept()
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
			app.console.Err("failedNsHandshake", err)
			continue
		}
		if !app.keys.IsValid(newClient.Path) {
			app.console.Err("invalidKeyNs[" + newClient.Path + "]")
			continue
		}
		client := app.stream.openNs(newClient)
		app.readNs(newClient)
		app.stream.closeNs(client)
	}
}
