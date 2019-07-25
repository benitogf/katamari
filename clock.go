package samo

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (app *Server) getTime() string {
	now := time.Now().UTC().UnixNano()
	return strconv.FormatInt(now, 10)
}

func (app *Server) sendTime() {
	go app.stream.broadcast(0, app.getTime(), true)
}

func (app *Server) tick() {
	ticker := time.NewTicker(app.Tick)
	for {
		select {
		case <-ticker.C:
			app.sendTime()
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
	go app.stream.write(client, app.getTime(), true)

	for {
		_, _, err := client.conn.ReadMessage()

		if err != nil {
			app.console.Err("readSocketError["+mode+"/"+key+"]", err)
			break
		}
	}
}
