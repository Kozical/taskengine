package engine

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/Kozical/taskengine/core"
)

type Dialer func() (net.Conn, error)
type ConnPool struct {
	pool    chan net.Conn
	minimum int
	dialer  Dialer
	quit    chan bool
}

func NewConnPool(min, max int, dialer Dialer) (pool ConnPool) {
	if min < 1 || max < 1 {
		panic("min and max must be greater than zero")
	}
	pool = ConnPool{
		pool:    make(chan net.Conn, max),
		quit:    make(chan bool, 1),
		dialer:  dialer,
		minimum: min,
	}
	pool.init()
	go pool.maintain()
	return
}

func (c ConnPool) Pop() net.Conn {
	return <-c.pool
}

func (c ConnPool) Push(conn net.Conn) {
	if len(c.pool) == cap(c.pool) {
		conn.Close()
		return
	}
	c.pool <- conn
}

func (c ConnPool) Close() {
	for len(c.pool) > 0 {
		conn := c.Pop()
		_ = conn.Close()
	}
	close(c.quit)
	close(c.pool)
}

func (c ConnPool) init() {
	for len(c.pool) < cap(c.pool) {
		conn, err := c.dialer()
		if err == nil {
			c.Push(conn)
		}
	}
}

func (c ConnPool) maintain() {
	for {
		select {
		case <-c.quit:
			return
		default:
			if len(c.pool) < c.minimum {
				conn, err := c.dialer()
				if err == nil {
					c.Push(conn)
				}
			}
			// Release the CPU for a few milliseconds
			time.Sleep(10 * time.Millisecond)
		}
	}
}

type RPCConfig struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	KeyPath string `json:"key_path"`
	CrtPath string `json:"crt_path"`
	CAPath  string `json:"ca_path"`
}

type RPCClient struct {
	pool     ConnPool
	quit     chan bool
	ticker   *time.Ticker
	endpoint string
	muPing   sync.Mutex
	errors   int64
	lastPong time.Time
}

var ErrorZeroAttempts = errors.New("Zero")

func NewRPCClient(min, max int, endpoint string, tlsConfig *tls.Config) (clt *RPCClient) {
	clt = &RPCClient{
		endpoint: endpoint,
		quit:     make(chan bool, 1),
		lastPong: time.Now(),
		pool: NewConnPool(
			min,
			max,
			func() (net.Conn, error) {
				return tls.Dial("tcp", endpoint, tlsConfig)
			}),
	}
	go clt.Heartbeat()
	return
}

func (r RPCClient) Call(method string, args interface{}, reply interface{}) (err error) {
	// Retry on : io.ErrUnexpectedEOF
	// Cache on : rpc.ErrShutdown
	err = ErrorZeroAttempts
	for i := 3; i > 0 && (err == io.ErrUnexpectedEOF || err == ErrorZeroAttempts); i-- {
		conn := r.pool.Pop()
		client := rpc.NewClient(conn)
		err = client.Call(method, args, reply)
		conn.Close()
	}
	return
}

func (r RPCClient) Endpoint() string {
	return r.endpoint
}

func (r RPCClient) Ready() bool {
	r.muPing.Lock()
	lastPong := r.lastPong
	errors := r.errors
	r.muPing.Unlock()

	// Make some better decisions for when a runner becomes unhealthy

	if time.Now().Sub(lastPong) > 5*time.Second {
		return false
	}
	if errors > 5 {
		return false
	}
	return true
}

func (r RPCClient) Heartbeat() {
	r.ticker = time.NewTicker(1 * time.Second)
	request := []byte("Ping!")
	for {
		select {
		case <-r.quit:
			return
		case <-r.ticker.C:
			var reply []byte
			err := r.Call("RPCTask.Ping", &request, &reply)
			if err != nil {
				r.muPing.Lock()
				r.errors++
				r.muPing.Unlock()
				continue
			}
			r.muPing.Lock()
			r.errors = 0
			r.lastPong = time.Now()
			r.muPing.Unlock()
		}
	}
}

func (r RPCClient) Close() {
	close(r.quit)
	r.pool.Close()
}

type RPCMgr struct {
	muClients    sync.Mutex
	Clients      []*RPCClient
	ReadyClients []*RPCClient

	Assignments map[*RPCClient][]*core.RPCJob

	muNextClient sync.Mutex
	nextClient   int
}

func NewRPCMgr(poolsize int, endpoints []string, tlsConfig *tls.Config) (mgr *RPCMgr) {
	mgr = &RPCMgr{
		Assignments: make(map[*RPCClient][]*core.RPCJob),
	}
	for _, e := range endpoints {
		log.Printf("Connecting to %s\n", e)
		c := NewRPCClient(poolsize, poolsize, e, tlsConfig)

		mgr.muClients.Lock()
		mgr.Clients = append(mgr.Clients, c)
		mgr.ReadyClients = append(mgr.Clients, c)
		mgr.muClients.Unlock()

	}
	return
}

func (mgr *RPCMgr) NextClient() (clt *RPCClient) {
	// boring round-robin.. maybe we can do better?
	// probably also only want to assign RPCClients that are 'Ready()'
	mgr.muNextClient.Lock()
	mgr.muClients.Lock()
	defer mgr.muClients.Unlock()
	defer mgr.muNextClient.Unlock()

	if len(mgr.ReadyClients) == 0 {
		return nil
	}

	failures := 0
	for {
		clt = mgr.ReadyClients[mgr.nextClient%len(mgr.ReadyClients)]
		mgr.nextClient++

		if clt.Ready() || failures > 3 {
			break
		}
		failures++
	}
	return
}

func (mgr *RPCMgr) MoveJobs(client *RPCClient) {
	if client.Ready() {
		return
	}
	for i, c := range mgr.ReadyClients {
		if c == client {
			mgr.muClients.Lock()
			mgr.ReadyClients = append(mgr.ReadyClients[:i], mgr.ReadyClients[i+1:]...)
			mgr.muClients.Unlock()
		}
	}
	for _, assignment := range mgr.Assignments[client] {
		mgr.DispatchJob(assignment)
	}
}

func (mgr *RPCMgr) DispatchJob(job *core.RPCJob) (err error) {
	var buf []byte
	client := mgr.NextClient()
	if client == nil {
		err = errors.New("NextClient() was nil")
		return
	}
	err = client.Call("RPCTask.Dispatch", job, &buf)
	if err != nil {
		log.Printf("Failed to dispatch job %s -> %v\n", job.Name, err)
		return
	}
	mgr.Assignments[client] = append(mgr.Assignments[client], job)
	return
}

func (mgr *RPCMgr) DispatchJobs(jobs []*core.RPCJob) (err error) {
	for _, j := range jobs {
		err = mgr.DispatchJob(j)
		if err != nil {
			return
		}
	}
	return
}
func (mgr *RPCMgr) Cleanup() {
	for _, c := range mgr.Clients {
		c.Close()
	}
}
