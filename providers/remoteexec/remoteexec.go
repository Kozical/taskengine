package remoteexec

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os"
	"strings"

	"github.com/Kozical/taskengine/job"
)

var config RemoteExecConfig
var tlsConfig *tls.Config

type RemoteExecConfig struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	KeyPath string `json:"key_path"`
	CrtPath string `json:"crt_path"`
	CAPath  string `json:"ca_path"`
}

/*
type Provider interface {
	Execute(*Job) (StateObject, error)
	Register(*Job, json.RawMessage) error
	New() Provider
	Name() string
	Cleanup()
}
*/

type RemoteExecState struct {
	Output string
}

func (res RemoteExecState) GetProperty(property string) interface{} {
	if property == "Output" {
		return res.Output
	}
	return ""
}

// RemoteExecProvider: implements job.Provider
type RemoteExecProvider struct {
	Settings struct {
		File string   `json:"File"`
		Args []string `json:"Args"`
	}
}

type APIRequest struct {
	File   string
	Params []string
}

type PowerShellResponse struct {
	Error string          `json:"error"`
	Data  json.RawMessage `json:"data"`
}

func NewRemoteExecProvider(path string) (rep *RemoteExecProvider, err error) {
	rep = new(RemoteExecProvider)
	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		return
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(&config)
	if err != nil {
		return
	}

	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair(config.CrtPath, config.KeyPath)
	if err != nil {
		return
	}

	pool := x509.NewCertPool()

	var b []byte
	b, err = ioutil.ReadFile(config.CAPath)
	if err != nil {
		return
	}

	pool.AppendCertsFromPEM(b)
	tlsConfig = &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
	}

	return
}

func (rep *RemoteExecProvider) Cleanup() {

}

func (rep *RemoteExecProvider) Name() string {
	return "remoteexec"
}

func (rep *RemoteExecProvider) Register(j *job.Job, raw json.RawMessage) (err error) {
	err = json.Unmarshal(raw, &rep.Settings)
	if err != nil {
		return
	}
	if len(rep.Settings.File) == 0 {
		err = errors.New("File parameter not provided to RemoteExec")
		return
	}
	if len(rep.Settings.Args) == 0 {
		err = errors.New("Args parameter not provided to RemoteExec")
		return
	}
	return
}

func (rep *RemoteExecProvider) New() job.Provider {
	return &RemoteExecProvider{}
}

func (rep *RemoteExecProvider) Execute(j *job.Job) (s job.StateObject, err error) {
	var conn *tls.Conn
	conn, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", config.Addr, config.Port), tlsConfig)
	if err != nil {
		return
	}
	defer conn.Close()

	req := &APIRequest{
		File:   rep.Settings.File,
		Params: rep.Settings.Args,
	}

	client := rpc.NewClient(conn)

	var buf []byte
	err = client.Call("API.Execute", req, &buf)
	if err != nil {
		return
	}

	s = RemoteExecState{
		Output: strings.Trim(string(buf), "\r\n"),
	}

	return
}
