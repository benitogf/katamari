package katamari

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Time returns a string timestamp
func (app *Server) Time() string {
	now := time.Now().UTC().UnixNano()
	return strconv.FormatInt(now, 10)
}

func (app *Server) sendTime() {
	go app.Stream.BroadcastTime(app.Time())
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

	client, err := app.Stream.New("", "", w, r)
	if err != nil {
		return
	}

	go app.Stream.WriteTime(client, app.Time())
	app.Stream.Read("", "", client)
}
