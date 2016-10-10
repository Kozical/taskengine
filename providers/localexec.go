package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/Kozical/taskengine/job"
)

type LocalExecAction struct {
	File string `json:"File"`
	Args string `json:"Args"`
}

type LocalExecState struct {
	Output string
}

func (l LocalExecState) GetProperty(property string) string {
	if property == "Output" {
		return l.Output
	}
	return ""
}

func init() {
	fmt.Println("Registering localexec")
	job.RegisterActionProvider("localexec_action", LocalExecActionFunc)
}

func LocalExecActionFunc(properties json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	var settings LocalExecAction

	err = json.Unmarshal(properties, &settings)
	if err != nil {
		return
	}
	if len(settings.File) == 0 {
		err = errors.New("File parameter not provided to LocalExec")
		return
	}
	if len(settings.Args) == 0 {
		err = errors.New("Args parameter not provided to LocalExec")
		return
	}

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(settings.File, settings.Args)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Output: %s\n", stderr.String())
		return
	}

	s = LocalExecState{
		Output: stdout.String(),
	}
	return
}
