package job

import (
	"encoding/json"
)

type Provider interface {
	Execute(*Job) (StateObject, error)
	Register(*Job, json.RawMessage) error
	New() Provider
	Name() string
	Cleanup()
}

type TaskEngine struct {
	providers []Provider
	Jobs      []*Job
}

func NewTaskEngine() *TaskEngine {
	return &TaskEngine{}
}

func (te *TaskEngine) ParseJobs(path string) (err error) {
	te.Jobs, err = ParseJobsInDirectory(te, path)
	return
}

func (te *TaskEngine) Cleanup() {
	for _, j := range te.Jobs {
		for _, t := range j.Tasks {
			t.Provider.Cleanup()
		}
	}
}

func (te *TaskEngine) RegisterProvider(providers ...Provider) {
	for _, p := range providers {
		te.providers = append(te.providers, p)
	}
}

func (te *TaskEngine) GetProvider(name string) Provider {
	for _, p := range te.providers {
		if p.Name() == name {
			return p
		}
	}
	return nil
}
