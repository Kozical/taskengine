package localpowershell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Kozical/taskengine/job"
)

type LocalPowerShellState struct {
	Stderr string
	Stdout string
}

func (l LocalPowerShellState) GetProperty(property string) string {
	if property == "Stdout" {
		return l.Stdout
	} else if property == "Stderr" {
		return l.Stderr
	}
	return ""
}

type LocalPowerShellActionProvider struct {
	Settings struct {
		File   string            `json:"File"`
		Args   []string          `json:"Args"`
		Params map[string]string `json:"Params"`
	}
}

func NewLocalPowerShellActionProvider() *LocalPowerShellActionProvider {
	return &LocalPowerShellActionProvider{}
}

func (lap *LocalPowerShellActionProvider) Name() string {
	return "local_powershell_action"
}

func (lap *LocalPowerShellActionProvider) Cleanup() {

}

func (lap *LocalPowerShellActionProvider) Action(raw json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	err = json.Unmarshal(raw, &lap.Settings)
	if err != nil {
		return
	}
	if len(lap.Settings.File) == 0 {
		err = fmt.Errorf("File parameter not provided to %s", lap.Name())
		return
	}
	if len(lap.Settings.Args) == 0 {
		err = fmt.Errorf("Args parameter not provided to %s", lap.Name())
		return
	}

	var args []string

	args = append(args, lap.Settings.Args...)
	if len(lap.Settings.Params) > 0 {
		for k, v := range lap.Settings.Params {
			args = append(args, k)
			args = append(args, v)
		}
	}

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(lap.Settings.File, args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		return
	}

	s = LocalPowerShellState{
		Stderr: stderr.String(),
		Stdout: stdout.String(),
	}
	return
}
