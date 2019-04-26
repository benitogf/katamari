package samo

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (app *Server) sendTime(clients []*conn) {
	now := time.Now().UTC().UnixNano()
	data := strconv.FormatInt(now, 10)
	for _, client := range clients {
		go app.stream.write(client, data, true)
	}
}

func (app *Server) tick() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			poolIndex := app.stream.findPool("ws", "time")
			app.stream.mutex.RLock()
			if poolIndex != -1 {
				go app.sendTime(app.stream.pools[poolIndex].connections)
			}
			app.stream.mutex.RUnlock()
		}
	}
}

func (app *Server) clock(w http.ResponseWriter, r *http.Request) {
	mode := "ws"
	key := "time"

	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("samo: this request is not authorized"))
		app.console.Err("socketConnectionUnauthorized", key)
		return
	}

	client, _, err := app.stream.new(mode, key, w, r)

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
