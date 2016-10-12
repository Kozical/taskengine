package listener

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/Kozical/taskengine/job"
)

var config *ListenerConfig

type ListenerConfig struct {
	BindAddress string `json:"bind_addr"`
	BindPort    int    `json:"bind_port"`
	UseTLS      bool   `json:"use_tls"`
	KeyPath     string `json:"key_path"`
	CrtPath     string `json:"crt_path"`
}

// ListenerProvider: Implements the job.Provider interface
type ListenerProvider struct {
	Settings struct {
		Method   string            `json:"Method"`
		Path     string            `json:"Path"`
		Headers  map[string]string `json:"Headers"`
		Response string            `json:"Response"`
	}
	State *ListenerState
}

type ListenerState struct {
	w http.ResponseWriter
	r *http.Request
}

func (l ListenerState) GetProperty(property string) interface{} {

	switch property {
	case "w":
		return l.w
	case "r":
		return l.r
	}

	return nil
}

func NewListenerProvider(path string) (lp *ListenerProvider, err error) {
	lp = new(ListenerProvider)

	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		err = fmt.Errorf("%s opening configuration failed", lp.Name())
		return
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(&config)
	if err != nil {
		err = fmt.Errorf("%s reading configuration failed", lp.Name())
		return
	}
	go lp.Listen()

	return
}

func (lp *ListenerProvider) Name() string {
	return "listener"
}

func (lp *ListenerProvider) Cleanup() {
}

func (lp *ListenerProvider) New() job.Provider {
	return &ListenerProvider{}
}

func (lp *ListenerProvider) Register(j *job.Job, raw json.RawMessage) (err error) {
	err = json.Unmarshal(raw, &lp.Settings)
	if err != nil {
		return
	}
	if len(lp.Settings.Method) == 0 {
		err = errors.New("Method parameter must be provided to Listener Provider!")
		return
	}
	if lp.Settings.Method != "Listen" {
		return nil
	}
	if len(lp.Settings.Path) == 0 {
		err = errors.New("Path parameter not provided to Listener Provider!")
		return
	}
	http.HandleFunc(lp.Settings.Path, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Request: %v\n", r.URL.String())
		lp.State = &ListenerState{
			w: w,
			r: r,
		}
		j.Run(lp)
	})
	return nil
}

func (lp *ListenerProvider) Execute(j *job.Job) (job.StateObject, error) {
	switch lp.Settings.Method {
	case "Listen":
		return lp.State, nil
	case "Respond":
		fmt.Printf("Calling respond\n")
		return lp.Respond(j)
	}
	return nil, fmt.Errorf("Method not found %s", lp.Settings.Method)
}

func (lp *ListenerProvider) Respond(j *job.Job) (s job.StateObject, err error) {
	for _, t := range j.Tasks {
		if t.Provider.Name() == lp.Name() && t.Provider != lp {
			lp.State = t.State.(*ListenerState)
		}
	}

	if lp.State == nil {
		err = errors.New("ListenerProvider Respond can only be used with the corresponding Listen provider")
		return
	}
	response := j.InterpolateState(lp.Settings.Response)

	fmt.Printf("Sending response: %s\n", response)

	for k, v := range lp.Settings.Headers {
		lp.State.w.Header().Add(k, v)
	}
	var n int
	n, err = lp.State.w.Write([]byte(response))
	if err != nil {
		return
	}
	if n < len(response) {
		err = fmt.Errorf("Write zero bytes to http.ResponseWriter, Response: %s\n", lp.Settings.Response)
		return
	}
	s = lp.State
	return
}

func (lp *ListenerProvider) Listen() {
	if config.UseTLS {
		err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", config.BindAddress, config.BindPort), config.CrtPath, config.KeyPath, nil)
		if err != nil {
			panic(fmt.Errorf("ListenAndServeTLS failed -> %v\n", err))
		}
	} else {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", config.BindAddress, config.BindPort), nil)
		if err != nil {
			panic(fmt.Errorf("ListenAndServe failed -> %v\n", err))
		}
	}
}
