package katamari

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/benitogf/cronexpr"
	"github.com/benitogf/katamari/objects"
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
			now := time.Now().UTC()
			for _, taskEntry := range app.tasks {
				// get tasks entries
				entry, err := app.Fetch(taskEntry.path+`/*`, taskEntry.path+`/*`)
				if err != nil {
					app.console.Err("failed to fetch task entry", err)
					continue
				}
				// decode cron entries for the task
				taskObjects, err := objects.DecodeList(entry.Data)
				if err != nil {
					app.console.Err("failed to decode task entry", err)
					continue
				}
				for _, taskStored := range taskObjects {
					taskObj, err := decodeTask([]byte(taskStored.Data))
					// parse stored cron expresion
					cron, err := cronexpr.Parse(taskObj.Cron)
					if err != nil {
						app.console.Err(err)
						continue
					}
					// evaluate cron expresion with current timestamp
					next := cron.Next(now)
					// app.console.Log(next.Format(time.RFC1123))
					// trigger the action of the task on mathed timestamps
					if next.Equal(now) {
						taskEntry.action(taskStored)
					}
				}
			}
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
