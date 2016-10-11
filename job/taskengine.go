package job

import (
	"encoding/json"
)

type EventProvider interface {
	Event(*Job, json.RawMessage, DispatchFunc) error
	Name() string
	Cleanup()
}
type ActionProvider interface {
	Action(json.RawMessage, *Job) (StateObject, error)
	Name() string
	Cleanup()
}
type TaskEngine struct {
	Events  []EventProvider
	Actions []ActionProvider
	Jobs    []*Job
}

func NewTaskEngine() *TaskEngine {
	return &TaskEngine{}
}

func (t *TaskEngine) ParseJobs(path string) (err error) {
	t.Jobs, err = ParseJobsInDirectory(t, path)
	return
}

func (t *TaskEngine) Cleanup() {
	for _, e := range t.Events {
		e.Cleanup()
	}
	for _, a := range t.Actions {
		a.Cleanup()
	}
}

func (t *TaskEngine) RegisterActionProvider(provider ...ActionProvider) {
	for _, p := range provider {
		t.Actions = append(t.Actions, p)
	}
}

func (t *TaskEngine) RegisterEventProvider(provider ...EventProvider) {
	for _, p := range provider {
		t.Events = append(t.Events, p)
	}
}

func (t *TaskEngine) GetEventProvider(name string) EventProvider {
	for _, e := range t.Events {
		if e.Name() == name {
			return e
		}
	}
	return nil
}

func (t *TaskEngine) GetActionProvider(name string) ActionProvider {
	for _, e := range t.Actions {
		if e.Name() == name {
			return e
		}
	}
	return nil
}
