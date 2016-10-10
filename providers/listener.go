package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Kozical/taskengine/job"
)

type (
	ListenerConfig struct {
		BindAddress string `json:"bind_addr"`
		BindPort    int    `json:"bind_port"`
		UseTLS      bool   `json:"use_tls"`
		KeyPath     string `json:"key_path"`
		CrtPath     string `json:"crt_path"`
	}

	ListenerEvent struct {
		Path string `json:"Path"`
	}

	ListenerAction struct {
		Headers  map[string]string `json:"Headers"`
		Response string            `json:"Response"`
	}

	ListenerState struct {
		w http.ResponseWriter
		r *http.Request
	}
)

func (l ListenerState) GetProperty(property string) string {
	values := l.r.URL.Query()

	v, ok := values[property]
	if ok {
		return v[0]
	}
	return ""
}

func init() {
	b, err := ioutil.ReadFile("config/listener.json")
	if err != nil {
		panic(err)
	}

	config := &ListenerConfig{}
	err = json.Unmarshal(b, config)
	if err != nil {
		panic(err)
	}

	go StartListener(config)
	fmt.Println("Registering listener")
	job.RegisterEventProvider("listener_event", ListenerEventFunc)
	job.RegisterActionProvider("listener_action", ListenerActionFunc)
}

func ListenerEventFunc(j *job.Job, properties json.RawMessage, dispatch job.DispatchFunc) error {
	var settings ListenerEvent
	err := json.Unmarshal(properties, &settings)
	if err != nil {
		fmt.Printf("RegisterListener failed to unmarshal json properties -> %v\n", err)
		return err
	}

	if len(settings.Path) == 0 {
		return errors.New("Path parameter not provided to Listener Provider!")
	}

	fmt.Printf("Registering Route: %s\n", settings.Path)
	http.HandleFunc(settings.Path, func(w http.ResponseWriter, r *http.Request) {
		state := ListenerState{
			w: w,
			r: r,
		}
		fmt.Printf("Listener Dispatching with state: %v\n", state)
		dispatch(state)
	})
	return nil
}

func ListenerActionFunc(raw json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	var settings ListenerAction

	err = json.Unmarshal(raw, &settings)
	if err != nil {
		return
	}

	state := j.Event.State.(ListenerState)
	for k, v := range settings.Headers {
		state.w.Header().Add(k, v)
	}
	state.w.Write([]byte(settings.Response))

	s = state
	return
}

func StartListener(config *ListenerConfig) {
	if config.UseTLS {
		err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", config.BindAddress, config.BindPort), config.CrtPath, config.KeyPath, nil)
		if err != nil {
			fmt.Printf("ListenAndServeTLS failed -> %v\n", err)
		}
	} else {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", config.BindAddress, config.BindPort), nil)
		if err != nil {
			fmt.Printf("ListenAndServe failed -> %v\n", err)
		}
	}
}
