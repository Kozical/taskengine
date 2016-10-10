package localexec

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"

	"github.com/Kozical/taskengine/job"
)

type LocalExecState struct {
	Output string
}

func (l LocalExecState) GetProperty(property string) string {
	if property == "Output" {
		return l.Output
	}
	return ""
}

// LocalExecActionProvider: implements job.ActionProvider
type LocalExecActionProvider struct {
	Settings struct {
		File string `json:"File"`
		Args string `json:"Args"`
	}
}

func NewLocalExecActionProvider() *LocalExecActionProvider {
	return &LocalExecActionProvider{}
}

func (lap *LocalExecActionProvider) Cleanup() {

}

func (lap *LocalExecActionProvider) Name() string {
	return "localexec_action"
}

func (lap *LocalExecActionProvider) Action(raw json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	err = json.Unmarshal(raw, &lap.Settings)
	if err != nil {
		return
	}
	if len(lap.Settings.File) == 0 {
		err = errors.New("File parameter not provided to LocalExec")
		return
	}
	if len(lap.Settings.Args) == 0 {
		err = errors.New("Args parameter not provided to LocalExec")
		return
	}

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(lap.Settings.File, lap.Settings.Args)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		return
	}

	s = LocalExecState{
		Output: stdout.String(),
	}
	return
}
