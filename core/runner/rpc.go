package runner

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"sync/atomic"
	"syscall"

	"github.com/Kozical/taskengine/core"
)

type RPCConfig struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	KeyPath string `json:"key_path"`
	CrtPath string `json:"crt_path"`
	CAPath  string `json:"ca_path"`
}

type RPCExec struct {
	File string
	Args []string
}

type RPCServer struct {
	quit  chan bool
	count int64
}

func NewRPCServer(receiver ...interface{}) (srv RPCServer, err error) {
	for _, r := range receiver {
		err = rpc.Register(r)
		if err != nil {
			return
		}
	}
	srv = RPCServer{
		quit: make(chan bool),
	}
	return
}

func (r RPCServer) Close() {
	close(r.quit)
}
func (r RPCServer) ListenAndServeTLS(addr string, config *tls.Config) (err error) {
	log.Printf("Listening for incoming connections on %s\n", addr)
	var lst net.Listener
	lst, err = tls.Listen("tcp", addr, config)
	if err != nil {
		return
	}
	defer lst.Close()

	for {
		var conn net.Conn
		conn, err = lst.Accept()
		if err != nil {
			fmt.Errorf("Error while accepting connection -> %v\n", err)
		}
		select {
		case <-r.quit:
			return
		default:
		}
		go func(c net.Conn) {
			atomic.AddInt64(&r.count, 1)
			log.Printf("%s accepted\n", c.RemoteAddr().String())
			defer func() {
				atomic.AddInt64(&r.count, -1)
				log.Printf("%s closed\n", c.RemoteAddr().String())
			}()
			rpc.ServeConn(c)
		}(conn)
	}
}

type RPCTask struct {
	T *Runner
}

func (r RPCTask) Ping(req *[]byte, res *[]byte) (err error) {
	if string(*req) == "Ping!" {
		*res = []byte("Pong!")
	}
	return
}

func (r RPCTask) Dispatch(j *core.RPCJob, res *[]byte) (err error) {
	var tasks []Task
	log.Printf("Dispatching job %s\n", j.Name)
	for _, v := range j.Objects {
		base := r.T.GetProvider(v.Provider)
		if base == nil {
			err = fmt.Errorf("Provider %s not found\n", v.Provider)
			return
		}
		var provider Provider
		provider = reflect.New(reflect.ValueOf(base).Elem().Type()).Interface().(Provider)
		tasks = append(tasks, Task{
			Title:      v.Name,
			Properties: core.JSONPromote(v.Properties),
			Provider:   provider,
		})
	}
	factory := JobFactory()

	for i, t := range tasks {
		if event, ok := t.Provider.(EventProvider); ok {
			go event.Register(factory(i, tasks))
		}
	}
	return
}

func (r RPCTask) Execute(req *RPCExec, res *[]byte) (err error) {
	log.Printf("Executing process %s\n", req.File)
	defer log.Printf("Execution completed: %s\n", req.File)

	var buf bytes.Buffer
	cmd := exec.Command(req.File, req.Args...)
	cmd.Stdout = &buf

	err = cmd.Start()
	if err != nil {
		return
	}

	intC := make(chan os.Signal)
	waitC := make(chan error, 1)

	signal.Notify(intC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		waitC <- cmd.Wait()
	}()

	for {
		select {
		case <-intC:
			cmd.Process.Kill()
			err = fmt.Errorf("Killed process %s\n", req.File)
			return
		case err = <-waitC:
			*res = buf.Bytes()
			return
		}
	}
}
