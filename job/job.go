package job

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

//type EventFunc func(*Job, json.RawMessage, DispatchFunc) error
//type ActionFunc func(json.RawMessage, *Job) (StateObject, error)

type StateObject interface {
	GetProperty(string) interface{}
}

type Task struct {
	Index      int
	Title      string
	Properties json.RawMessage
	State      StateObject
	Provider   Provider
}

type Job struct {
	Name  string
	Tasks []*Task
}

func (j *Job) GetProperty(title, property string) interface{} {
	for _, t := range j.Tasks {
		if t.Title == title {
			return t.State.GetProperty(property)
		}
	}
	return nil
}

func (j *Job) Register() (err error) {
	for _, t := range j.Tasks {
		err = t.Provider.Register(j, t.Properties)
		if err != nil {
			return
		}
	}
	return
}
func (j *Job) GetStateByResourceTitle(name string) StateObject {
	for _, t := range j.Tasks {
		if t.Title == name {
			return t.State
		}
	}
	return nil
}

func (j *Job) Run(provider Provider) (err error) {
	var match bool
	for _, t := range j.Tasks {
		if provider == t.Provider {
			match = true
		}
		if match {
			var state StateObject
			state, err = t.Provider.Execute(j)
			if err != nil {
				fmt.Printf("Error while executing %s::%s -> %v\n", t.Provider.Name(), t.Title, err)
				break
			}
			t.State = state
		}
	}
	return nil
}

type stateParser struct {
	input  string
	output bytes.Buffer
	pos    int
	lpos   int
	width  int
	start  int
	exps   int
}

func (s *stateParser) next() rune {
	if s.pos >= len(s.input) {
		return -1
	}
	r, w := utf8.DecodeRuneInString(s.input[s.pos:])
	s.width = w
	s.lpos = s.pos
	s.pos += s.width
	return r
}

func (s *stateParser) getStateProperty(exp string, j *Job) (property string, ok bool) {
	parts := strings.Split(exp, ".")
	if len(parts) > 1 {
		state := j.GetStateByResourceTitle(parts[0])
		if state != nil {
			ok = true
			property = state.GetProperty(strings.Join(parts[1:], ".")).(string)
		}
	}
	return
}

func (s *stateParser) replaceExpression(j *Job) {
	property, ok := s.getStateProperty(s.input[s.start+2:s.pos-1], j)
	if !ok {
		s.output.WriteString(s.input[s.start:s.pos])
		return
	}
	s.output.WriteString(property)
}
func (j *Job) InterpolateState(data string) string {
	var s stateParser
	s.input = data
	var insideExpression bool
	for {
		r := s.next()
		if r == -1 {
			break
		}
		switch {
		case strings.HasPrefix(s.input[s.lpos:], "$("):
			insideExpression = true
			s.start = s.lpos
		case r == ')' && insideExpression:
			s.replaceExpression(j)
			insideExpression = false
		case !insideExpression:
			s.output.WriteRune(r)
		}
	}
	return s.output.String()
}

/*
// interpolateState: takes a string that potentially has $(state.something) in it
// and tries to expand into state (map[string]interface{}) as deep as necessary
// expects that the resultant value is always printable as a string
// the interpolated value is inlined in the original context of the string
// if there is not an expression in the string, then the original contents are
// returned as a new string
//$( <resource_title>.<property_name> )
func (j *Job) InterpolateState(data []byte) []byte {
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
*/
