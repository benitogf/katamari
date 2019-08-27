package katamari

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
	go app.Stream.BroadcastTime(app.getTime())
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
	if !app.Audit(r) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("katamari: this request is not authorized"))
		app.console.Err("socketConnectionUnauthorized time")
		return
	}

	client, _, err := app.Stream.New("", w, r)

	if err != nil {
		return
	}

	go app.Stream.WriteTime(client, app.getTime())
	app.Stream.Read("", client)
}
