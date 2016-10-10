package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/Kozical/taskengine/job"
)

type LocalPowerShellAction struct {
	File   string            `json:"File"`
	Args   []string          `json:"Args"`
	Params map[string]string `json:"Params"`
}

type LocalPowerShellState struct {
	Output string
}

func (l LocalPowerShellState) GetProperty(property string) string {
	if property == "Output" {
		return l.Output
	}
	return ""
}

func init() {
	fmt.Println("Registering localpowershell")
	job.RegisterActionProvider("local_powershell_action", LocalPowerShellActionFunc)
}

func PowerShellEscape(data string) string {
	/*
		`0 - null            - 0
		`a - alert           - 7
		`b - backspace       - 8
		`f - form feed       - 12
		`n - line feed       - 10
		`r - carriage return - 13
		`t - horizontal tab  - 9
		`v - vertical tab    - 11
		`` - grave character - 96
		`# - octothorpe      - 35
		`' - single quote    - 39
		`" - double quote    - 34
	*/
	var buf bytes.Buffer

	for _, v := range data {
		switch v {
		case 0:
			buf.WriteString("`0")
		case 7:
			buf.WriteString("`a")
		case 8:
			buf.WriteString("`b")
		case 9:
			buf.WriteString("`t")
		case 10:
			buf.WriteString("`n")
		case 11:
			buf.WriteString("`v")
		case 12:
			buf.WriteString("`f")
		case 13:
			buf.WriteString("`r")
		case 34:
			buf.WriteString("`\"")
		case 35:
			buf.WriteString("`#")
		case 39:
			buf.WriteString("`'")
		case 96:
			buf.WriteString("``")
		default:
			buf.WriteRune(v)

		}
	}
	return buf.String()
}

func LocalPowerShellActionFunc(properties json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	var settings LocalPowerShellAction

	err = json.Unmarshal(properties, &settings)
	if err != nil {
		return
	}
	if len(settings.File) == 0 {
		err = errors.New("File parameter not provided to LocalPowerShell")
		return
	}
	if len(settings.Args) == 0 {
		err = errors.New("Args parameter not provided to LocalPowerShell")
		return
	}

	var args []string

	args = append(args, settings.Args...)
	if len(settings.Params) > 0 {
		for k, v := range settings.Params {
			args = append(args, k)
			args = append(args, v)
		}
	}

	fmt.Printf("Arguments: %v\n", args)

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(settings.File, args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Output: %s\n", stderr.String())
		return
	}

	s = LocalPowerShellState{
		Output: stdout.String(),
	}
	return
}
