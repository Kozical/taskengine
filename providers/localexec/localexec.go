package localexec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/Kozical/taskengine/core/runner"
)

// LocalExecActionProvider: implements core.ActionProvider
type LocalExecProvider struct {
	Properties map[string]string
	Settings   struct {
		File string   `json:"File"`
		Args []string `json:"Args"`
	}
}

func NewLocalExecProvider() *LocalExecProvider {
	return &LocalExecProvider{}
}

func (lp *LocalExecProvider) String() string {
	return fmt.Sprintf("LocalExecProvider{Properties: %v}\n", lp.Properties)
}

/*
type Provider interface {
	Execute(*Job) (err error)
}
*/

func (lp *LocalExecProvider) Execute(j *runner.Job) (err error) {

	fmt.Println("localexec:", lp, "job:", j)

	var task *runner.Task
	for _, t := range j.Tasks {
		if t.Provider == lp {
			task = &t
			break
		}
	}
	if task == nil {
		err = errors.New("LocalExecProvider Task was nil")
		return
	}

	properties := j.InterpolateState(string(task.Properties))

	err = json.Unmarshal(properties, &lp.Settings)
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
	lp.Properties = make(map[string]string)

	for _, name := range []string{"Stdout", "Stderr"} {
		lp.Properties[name] = fmt.Sprintf("%s.%s", task.Title, name)
	}

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(lp.Settings.File, lp.Settings.Args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("Error executing %s -> %v\n", lp.Settings.File, err)
		return
	}

	fmt.Printf("localexec executed: %s %s result: %s\n", lp.Settings.File, lp.Settings.Args, stdout.String())

	j.State[lp.Properties["Stdout"]] = func() interface{} { return stdout.String() }
	j.State[lp.Properties["Stderr"]] = func() interface{} { return stderr.String() }
	return
}
