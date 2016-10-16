package listener

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	//	"reflect"
	//	"sync"

	"github.com/Kozical/taskengine/core/runner"
)

// ListenerProvider: Implements the core.Provider interface
type ListenerProvider struct {
	Title      string
	Properties map[string]string
	Config     struct {
		BindAddress string `json:"bind_addr"`
		BindPort    int    `json:"bind_port"`
		UseTLS      bool   `json:"use_tls"`
		KeyPath     string `json:"key_path"`
		CrtPath     string `json:"crt_path"`
	}
	Settings struct {
		Method   string            `json:"Method"`
		Path     string            `json:"Path"`
		Headers  map[string]string `json:"Headers"`
		Response string            `json:"Response"`
	}
}

func NewListenerProvider(path string) (lp *ListenerProvider, err error) {
	lp = &ListenerProvider{}

	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		err = fmt.Errorf("ListenerProvider opening configuration failed")
		return
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(&lp.Config)
	if err != nil {
		err = fmt.Errorf("ListenerProvider reading configuration failed")
		return
	}
	go lp.Listen()
	return
}

func (lp *ListenerProvider) String() string {
	return fmt.Sprintf("ListenerProvider{Title: %s, Properties: %v}\n", lp.Title, lp.Properties)
}

func (lp *ListenerProvider) Execute(j *runner.Job) error {
	switch lp.Settings.Method {
	case "Respond":
		return lp.Respond(j)
	case "Listen":
		return nil
	default:
		return errors.New("Method not implemented in ListenerProvider")
	}
}

func (lp *ListenerProvider) Register(fn func() *runner.Job) {
	fmt.Println("ListenerProvider Register() called")
	job := fn()

	var task *runner.Task
	for _, t := range job.Tasks {
		if t.Provider == lp {
			task = &t
			break
		}
	}
	if task == nil {
		fmt.Printf("ListenerProvider.Register() task was nil")
		return
	}
	fmt.Println(task, lp)
	err := json.Unmarshal(task.Properties, &lp.Settings)
	if err != nil {
		fmt.Printf("Failed to unmarshal ListenerProvider Properties\n")
		return
	}
	if len(lp.Settings.Method) == 0 {
		fmt.Printf("Method parameter must be provided to Listener Provider!\n")
		return
	}
	lp.Properties = make(map[string]string)

	for _, name := range []string{"W", "R", "Closer"} {
		lp.Properties[name] = fmt.Sprintf("%s.%s", task.Title, name)
	}

	lp.Title = task.Title

	switch lp.Settings.Method {
	case "Listen":
		lp.RegisterListen(task, fn)
	case "Respond":
		lp.RegisterRespond(task, fn)
	default:
		return
	}
}

func (lp *ListenerProvider) RegisterRespond(t *runner.Task, fn func() *runner.Job) {

}
func (lp *ListenerProvider) RegisterListen(t *runner.Task, fn func() *runner.Job) {
	if len(lp.Settings.Path) == 0 {
		fmt.Printf("Path parameter not provided to Listener Provider!")
		return
	}
	fmt.Println("ListenerProvider RegisterListener() called")
	http.HandleFunc(lp.Settings.Path, func(w http.ResponseWriter, r *http.Request) {
		j := fn()
		closer := make(chan struct{}, 0)
		for k, _ := range r.URL.Query() {
			j.Store(fmt.Sprintf("%s.URL.%s", lp.Title, k), func() interface{} {
				key := k
				return j.State[lp.Properties["R"]]().(*http.Request).URL.Query()[key][0]
			})
		}

		j.Store(lp.Properties["W"], func() interface{} { return w })
		j.Store(lp.Properties["R"], func() interface{} { return r })
		j.Store(lp.Properties["Closer"], func() interface{} {
			closer <- struct{}{}
			return nil
		})
		fmt.Printf("Request Received: %v\n", r.URL.String())

		j.Run()
		<-closer
	})
}

func (lp *ListenerProvider) Respond(j *runner.Job) (err error) {
	w := j.State[lp.Properties["W"]]().(http.ResponseWriter)

	response := j.InterpolateState(lp.Settings.Response)

	fmt.Printf("Sending response: %s\n", response)

	for k, v := range lp.Settings.Headers {
		w.Header().Add(k, v)
	}

	_, err = w.Write(response)

	j.State[lp.Properties["Closer"]]()
	return
}

func (lp *ListenerProvider) Listen() {
	fmt.Printf("Listening on 8081\n")
	if lp.Config.UseTLS {
		err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", lp.Config.BindAddress, lp.Config.BindPort), lp.Config.CrtPath, lp.Config.KeyPath, nil)
		if err != nil {
			panic(fmt.Errorf("ListenAndServeTLS failed -> %v\n", err))
		}
	} else {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", lp.Config.BindAddress, lp.Config.BindPort), nil)
		if err != nil {
			panic(fmt.Errorf("ListenAndServe failed -> %v\n", err))
		}
	}
}
