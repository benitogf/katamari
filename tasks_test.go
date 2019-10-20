package katamari

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/stretchr/testify/require"
)

func TestTask(t *testing.T) {
	t.Skip() // this test can take up to 59 seconds to pass, so better to skip it
	// TODO: add seconds support to the cronexpr parser?
	var wg sync.WaitGroup
	var app = Server{}
	app.Silence = true
	app.Tick = 1 * time.Second // ticks bellow 1sec or above 59sec will cause multi executions or missed executions
	app.Task("writer", func(data objects.Object) {
		app.console.Log(data)
		wg.Done()
	})
	app.Start("localhost:0")
	defer app.Close(os.Interrupt)
	taskData := messages.Encode([]byte(`{
		"cron": "* * * * *",
		"extra": "write something 🧰"
	}`))
	var writerTask = []byte(`{"data":"` + taskData + `"}`)
	req := httptest.NewRequest("POST", "/writer/*", bytes.NewBuffer(writerTask))
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)
	wg.Add(1)
	wg.Wait()
}
