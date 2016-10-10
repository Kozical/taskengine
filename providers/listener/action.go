package listener

import (
	"encoding/json"
	"fmt"

	"github.com/Kozical/taskengine/job"
)

// ListenerActionProvider: Implements the job.ActionProvider interface
type ListenerActionProvider struct {
	Settings struct {
		Headers  map[string]string `json:"Headers"`
		Response string            `json:"Response"`
	}
}

func NewListenerActionProvider() *ListenerActionProvider {
	return &ListenerActionProvider{}
}

func (lap *ListenerActionProvider) Name() string {
	return "listener_action"
}

func (lap *ListenerActionProvider) Cleanup() {
}

func (lap *ListenerActionProvider) Action(raw json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	err = json.Unmarshal(raw, &lap.Settings)
	if err != nil {
		return
	}
	if j.Event.State == nil {
		err = fmt.Errorf("ListenerActionProvider(%s) can only be used with the corresponding event provider", lap.Name())
		return
	}
	state := j.Event.State.(ListenerState)
	for k, v := range lap.Settings.Headers {
		state.w.Header().Add(k, v)
	}
	var n int
	n, err = state.w.Write([]byte(lap.Settings.Response))
	if err != nil {
		return
	}
	if n < len(lap.Settings.Response) {
		err = fmt.Errorf("Write zero bytes to http.ResponseWriter, Response: %s\n", lap.Settings.Response)
		return
	}
	s = state
	return
}
