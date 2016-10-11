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

	"github.com/Kozical/taskengine/job"
)

type RemoteExecState struct {
	Output string
}

func (res RemoteExecState) GetProperty(property string) string {
	if property == "Output" {
		return res.Output
	}
	return ""
}

// RemoteExecActionProvider: implements job.ActionProvider
type RemoteExecActionProvider struct {
	configPath string
	tlsConfig  *tls.Config
	Config     struct {
		Addr    string `json:"addr"`
		Port    int    `json:"port"`
		KeyPath string `json:"key_path"`
		CrtPath string `json:"crt_path"`
		CAPath  string `json:"ca_path"`
	}
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

func NewRemoteExecActionProvider(path string) (r *RemoteExecActionProvider, err error) {
	r = &RemoteExecActionProvider{
		configPath: path,
	}
	err = r.init()
	return
}

func (rap *RemoteExecActionProvider) init() (err error) {
	var f *os.File
	f, err = os.Open(rap.configPath)
	if err != nil {
		return
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(&rap.Config)
	if err != nil {
		return
	}

	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair(rap.Config.CrtPath, rap.Config.KeyPath)
	if err != nil {
		return
	}

	pool := x509.NewCertPool()

	var b []byte
	b, err = ioutil.ReadFile(rap.Config.CAPath)
	if err != nil {
		return
	}

	pool.AppendCertsFromPEM(b)
	rap.tlsConfig = &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
	}

	return nil
}

func (rap *RemoteExecActionProvider) Cleanup() {

}

func (rap *RemoteExecActionProvider) Name() string {
	return "remoteexec_action"
}

func (rap *RemoteExecActionProvider) Action(raw json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	err = json.Unmarshal(raw, &rap.Settings)
	if err != nil {
		return
	}
	if len(rap.Settings.File) == 0 {
		err = errors.New("File parameter not provided to RemoteExec")
		return
	}
	if len(rap.Settings.Args) == 0 {
		err = errors.New("Args parameter not provided to RemoteExec")
		return
	}

	var conn *tls.Conn
	conn, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", rap.Config.Addr, rap.Config.Port), rap.tlsConfig)
	if err != nil {
		return
	}
	defer conn.Close()

	req := &APIRequest{
		File:   rap.Settings.File,
		Params: rap.Settings.Args,
	}

	client := rpc.NewClient(conn)

	var buf []byte
	err = client.Call("API.Execute", req, &buf)
	if err != nil {
		return
	}

	s = RemoteExecState{
		Output: string(buf),
	}

	return
}
