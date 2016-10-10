package listener

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/Kozical/taskengine/job"
)

type ListenerEventProvider struct {
	configPath string
	Config     struct {
		BindAddress string `json:"bind_addr"`
		BindPort    int    `json:"bind_port"`
		UseTLS      bool   `json:"use_tls"`
		KeyPath     string `json:"key_path"`
		CrtPath     string `json:"crt_path"`
	}
	Settings struct {
		Path string `json:"Path"`
	}
}

func NewListenerEventProvider(path string) (lep *ListenerEventProvider, err error) {
	lep = &ListenerEventProvider{
		configPath: path,
	}
	err = lep.init()
	return
}

func (lep *ListenerEventProvider) init() error {
	f, err := os.Open(lep.configPath)
	if err != nil {
		return fmt.Errorf("%s opening configuration failed", lep.Name())
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(&lep.Config)
	if err != nil {
		return fmt.Errorf("%s reading configuration failed", lep.Name())
	}
	go lep.Listen()
	return nil
}

func (lep *ListenerEventProvider) Cleanup() {

}

func (lep *ListenerEventProvider) Name() string {
	return "listener_event"
}
func (lep *ListenerEventProvider) Event(j *job.Job, raw json.RawMessage, dispatch job.DispatchFunc) error {
	err := json.Unmarshal(raw, &lep.Settings)
	if err != nil {
		return err
	}
	if len(lep.Settings.Path) == 0 {
		return errors.New("Path parameter not provided to Listener Provider!")
	}
	http.HandleFunc(lep.Settings.Path, func(w http.ResponseWriter, r *http.Request) {
		state := ListenerState{
			w: w,
			r: r,
		}
		dispatch(state)
	})
	return nil
}
func (lep *ListenerEventProvider) Listen() {
	if lep.Config.UseTLS {
		err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", lep.Config.BindAddress, lep.Config.BindPort), lep.Config.CrtPath, lep.Config.KeyPath, nil)
		if err != nil {
			panic(fmt.Errorf("ListenAndServeTLS failed -> %v\n", err))
		}
	} else {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", lep.Config.BindAddress, lep.Config.BindPort), nil)
		if err != nil {
			panic(fmt.Errorf("ListenAndServe failed -> %v\n", err))
		}
	}
}
