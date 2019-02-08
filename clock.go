package samo

import (
	"net/http"
	"strconv"
	"time"
)

func (app *Server) sendTime(clients []*conn) {
	now := time.Now().UTC().UnixNano()
	data := strconv.FormatInt(now, 10)
	for _, client := range clients {
		go app.stream.write(client, data)
	}
}

func (app *Server) tick() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			poolIndex := app.stream.findPool("ws", "time")
			if poolIndex != -1 {
				app.sendTime(app.stream.pools[poolIndex].connections)
			}
		}
	}
}

func (app *Server) clock(w http.ResponseWriter, r *http.Request) {
	mode := "ws"
	key := "time"
	client, err := app.stream.new(mode, key, w, r)

	if err != nil {
		return
	}

	defer app.stream.close(mode, key, client)
	app.sendTime([]*conn{client})

	for {
		_, _, err := client.conn.ReadMessage()

		if err != nil {
			app.console.Err("readSocketError["+mode+"/"+key+"]", err)
			break
		}
	}
}
