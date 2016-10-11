package job

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type DispatchFunc func(StateObject) error

//type EventFunc func(*Job, json.RawMessage, DispatchFunc) error
//type ActionFunc func(json.RawMessage, *Job) (StateObject, error)

type StateObject interface {
	GetProperty(string) string
}

type Task struct {
	Title      string
	Properties json.RawMessage
	State      StateObject
}

type Action struct {
	Task
	Provider ActionProvider
}

type Event struct {
	Task
	Provider EventProvider
}

type Job struct {
	Name    string
	Event   *Event
	Actions []*Action
}

func (j *Job) GetStateByProvider(name string) StateObject {
	if j.Event.Provider.Name() == name {
		return j.Event.State
	}
	for _, a := range j.Actions {
		if a.Provider.Name() == name {
			return a.State
		}
	}
	return nil
}

func (j *Job) GetStateByResource(name string) StateObject {
	if j.Event.Title == name {
		return j.Event.State
	}
	for _, a := range j.Actions {
		if a.Title == name {
			return a.State
		}
	}
	return nil
}

func (j *Job) Register() (err error) {
	err = j.Event.Provider.Event(j, j.Event.Properties, j.Run)
	return
}

func (j *Job) Run(state StateObject) error {
	j.Event.State = state
	fmt.Printf("Job Dispatched: %s\n", j.Name)

	for _, v := range j.Actions {
		p := j.interpolateState(v.Properties)
		fmt.Printf("Calling Action[%s] -> %s\n", v.Provider.Name(), p)
		actionState, err := v.Provider.Action(p, j)
		if err != nil {
			return fmt.Errorf("Error in Action[%s] -> %v\n", v.Provider.Name(), err)
		}
		v.State = actionState
	}
	return nil
}

// interpolateState: takes a string that potentially has $(state.something) in it
// and tries to expand into state (map[string]interface{}) as deep as necessary
// expects that the resultant value is always printable as a string
// the interpolated value is inlined in the original context of the string
// if there is not an expression in the string, then the original contents are
// returned as a new string
//$( <resource_title>.<property_name> )
func (j *Job) interpolateState(data []byte) []byte {
	var insideExpression bool
	var out bytes.Buffer
	var exp bytes.Buffer
	for i := 0; i < len(data); i++ {
		if strings.HasPrefix(string(data[i:]), "$(") {
			i++
			insideExpression = true
			continue
		}
		if insideExpression && data[i] == ')' {
			insideExpression = false
			// inject new data
			if exp.Len() > 0 {
				var obj string
				parts := strings.Split(exp.String(), ".")
				if len(parts) > 1 {
					state := j.GetStateByResource(parts[0])
					fmt.Printf("State: %v\n", state)
					if state != nil {
						obj = JSONEscape(state.GetProperty(strings.Join(parts[1:], ".")))
						fmt.Printf("obj: %s\n", obj)
					}
				}
				out.WriteString(obj)
			}
			continue
		}
		if insideExpression {
			exp.WriteByte(data[i])
			continue
		}
		out.WriteByte(data[i])
	}
	return out.Bytes()
}
