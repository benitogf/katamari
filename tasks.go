package katamari

import (
	"encoding/json"

	"github.com/benitogf/katamari/objects"
)

type taskObject struct {
	Cron string `json:"cron"`
}

type perform func(data objects.Object)

type task struct {
	path   string
	action perform
}

type tasks []task

// Task will create a task to be performed by the server
//
// tasks have a storage base key associated
//
// Objects must include a cron expresion
//
// extra fields can be included in the object and will be passed to the action
//
// {
//
//		cron: '0 22 * * *',
//
//    someExtraField: 'extra information needed by the action'
//
// }
//
func (app *Server) Task(path string, action perform) {
	app.tasks = append(app.tasks, task{
		path:   path,
		action: action,
	})
	// TODO: create default cron filter to validate object format
	// TODO: allow filter param on task creation
	app.OpenFilter(path + `/*`)
}

func decodeTask(data []byte) (taskObject, error) {
	var obj taskObject
	err := json.Unmarshal(data, &obj)

	return obj, err
}
