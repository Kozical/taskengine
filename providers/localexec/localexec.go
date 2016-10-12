package localexec

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"

	"github.com/Kozical/taskengine/job"
)

type LocalExecState struct {
	Stderr string
	Stdout string
}

func (l LocalExecState) GetProperty(property string) interface{} {
	switch property {
	case "Stderr":
		return l.Stderr
	case "Stdout":
		return l.Stdout
	}
	return nil
}

// LocalExecActionProvider: implements job.ActionProvider
type LocalExecProvider struct {
	Settings struct {
		File string `json:"File"`
		Args string `json:"Args"`
	}
}

func NewLocalExecProvider() *LocalExecProvider {
	return &LocalExecProvider{}
}

func (lp *LocalExecProvider) Cleanup() {

}

func (lp *LocalExecProvider) Name() string {
	return "localexec"
}

func (lp *LocalExecProvider) New() job.Provider {
	return &LocalExecProvider{}
}

func (lp *LocalExecProvider) Register(j *job.Job, raw json.RawMessage) (err error) {
	err = json.Unmarshal(raw, &lp.Settings)
	if err != nil {
		return
	}
	if len(lp.Settings.File) == 0 {
		err = errors.New("File parameter not provided to LocalExec")
		return
	}
	if len(lp.Settings.Args) == 0 {
		err = errors.New("Args parameter not provided to LocalExec")
		return
	}
	return
}

func (lp *LocalExecProvider) Execute(j *job.Job) (s job.StateObject, err error) {

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(lp.Settings.File, lp.Settings.Args)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		return
	}

	s = LocalExecState{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	return
}
