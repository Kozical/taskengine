package job

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type DispatchFunc func(StateObject)
type EventFunc func(*Job, json.RawMessage, DispatchFunc) error
type ActionFunc func(json.RawMessage, *Job) (StateObject, error)

type StateObject interface {
	GetProperty(string) string
}

type Action struct {
	Name       string
	Provider   string
	Action     ActionFunc
	Properties json.RawMessage
	State      StateObject
}

type Event struct {
	Name       string
	Provider   string
	Event      EventFunc
	Properties json.RawMessage
	State      StateObject
}

type Job struct {
	Name    string
	Event   *Event
	Actions []*Action
}

func (j *Job) GetStateObject(name string) StateObject {
	if j.Event.Name == name {
		return j.Event.State
	}
	for _, a := range j.Actions {
		if a.Name == name {
			return a.State
		}
	}
	return nil
}

func (j *Job) Register() {
	fmt.Printf("Registering job %s\n", j.Name)
	err := j.Event.Event(j, j.Event.Properties, j.Run)
	if err != nil {
		fmt.Printf("Failed to register %s\n", j.Name)
	}
}

func (j *Job) Run(state StateObject) {
	j.Event.State = state

	for _, v := range j.Actions {
		p := j.interpolateState(v.Properties)
		fmt.Printf("Calling Action[%s] -> %s\n", v.Name, p)
		actionState, err := v.Action(p, j)
		if err != nil {
			fmt.Printf("Error in Action[%s] -> %v\n", v.Name, err)
			break
		}
		v.State = actionState
	}
}

// interpolateState: takes a string that potentially has $(state.something) in it
// and tries to expand into state (map[string]interface{}) as deep as necessary
// expects that the resultant value is always printable as a string
// the interpolated value is inlined in the original context of the string
// if there is not an expression in the string, then the original contents are
// returned as a new string
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
					state := j.GetStateObject(parts[0])
					if state != nil {
						obj = JSONEscape(state.GetProperty(strings.Join(parts[1:], ".")))
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

var RegisteredActionProvidersLock sync.Mutex
var RegisteredActionProviders map[string]ActionFunc

var RegisteredEventProvidersLock sync.Mutex
var RegisteredEventProviders map[string]EventFunc

func RegisterActionProvider(name string, f ActionFunc) {
	RegisteredActionProvidersLock.Lock()
	defer RegisteredActionProvidersLock.Unlock()

	if RegisteredActionProviders == nil {
		RegisteredActionProviders = make(map[string]ActionFunc)
	}

	RegisteredActionProviders[name] = f
}

func RegisterEventProvider(name string, f EventFunc) {
	RegisteredEventProvidersLock.Lock()
	defer RegisteredEventProvidersLock.Unlock()
	if RegisteredEventProviders == nil {
		RegisteredEventProviders = make(map[string]EventFunc)
	}
	RegisteredEventProviders[name] = f
}
