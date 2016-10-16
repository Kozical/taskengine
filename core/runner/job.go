package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Provider interface {
	Execute(*Job) error
}

type EventProvider interface {
	Register(func() *Job)
}

type Task struct {
	Title      string
	Properties json.RawMessage
	Provider   Provider
}

func (t Task) String() string {
	return fmt.Sprintf("Task{Title: %q, Properties: %q, Provider: %q}\n", t.Title, t.Properties, t.Provider)
}

type Job struct {
	ID    int
	State map[string]func() interface{}
	Tasks []Task
}

func (j *Job) String() string {
	return fmt.Sprintf("Job{ID: %d, State: %v, Tasks[%s]}\n", j.ID, j.State, j.Tasks)
}

func (j *Job) Store(key string, fn func() interface{}) {
	j.State[key] = fn
}

func (j *Job) Run() {
	go func() {
		for _, t := range j.Tasks {
			err := t.Provider.Execute(j)
			if err != nil {
				fmt.Printf("Error while executing %s -> %v\n", t.Title, err)
				break
			}
		}
		fmt.Println(j)
	}()
}

func JobFactory() func(int, []Task) func() *Job {
	id := 0
	return func(i int, tasks []Task) func() *Job {
		return func() *Job {
			j := &Job{id, make(map[string]func() interface{}), tasks[i:]}
			id += 1
			return j
		}
	}
}

/*

//type EventFunc func(*Job, json.RawMessage, DispatchFunc) error
//type ActionFunc func(json.RawMessage, *Job) (StateObject, error)

type StateObject interface {
	GetProperty(string) interface{}
}

type Task struct {
	Index      int
	Title      string
	Properties json.RawMessage
	//State      StateObject
	Provider Provider
}

type Job struct {
	Name  string
	Tasks []*Task
}

func (j *Job) GetProperty(title, property string, chain StateSegment) interface{} {
	var state StateSegment
	state = chain

	for state.Previous != nil {

		state = state.Previous
	}
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
		fmt.Printf("Task[%d]: %q != %q\n", t.Index, t.Title, name)
		if t.Title == name {
			return t.State
		}
	}
	return nil
}

func (j *Job) Run(provider Provider, initial StateSegment) (err error) {
	var match bool
	var chain StateSegment
	// listener ListenForConnections
	// localexec DoStuff
	// listener Respond
	chain = initial
	for i, t := range j.Tasks {
		if provider == t.Provider {
			match = true
			continue
		}
		if match {
			fmt.Printf("RUN[%d]: %s -> %s\n", i, t.Title, t.Properties)
			var state StateObject
			state, err = t.Provider.Execute(j, chain)
			if err != nil {
				fmt.Printf("Error while executing %s::%s -> %v\n", t.Provider.Name(), t.Title, err)
				break
			}
			fmt.Printf("Task: %s State: %v\n", t.Title, t.State)
			chain = StateSegment{
				Previous: &chain,
				State:    state,
			}
			//t.State = state
		}
	}
	return nil
}
*/
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
	fmt.Printf("Getting property %s\n", exp)
	if val, exists := j.State[exp]; exists {
		ok = true
		property = fmt.Sprint(val())
		return
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
func (j *Job) InterpolateState(data string) []byte {
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
	return s.output.Bytes()
}
