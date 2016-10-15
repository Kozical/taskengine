package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kozical/taskengine/core/runner"

	"github.com/Kozical/taskengine/providers/listener"
	"github.com/Kozical/taskengine/providers/localexec"
	"github.com/Kozical/taskengine/providers/mongo"
	"github.com/Kozical/taskengine/providers/remoteexec"
	"github.com/Kozical/taskengine/providers/ticker"
)

func main() {

	port := flag.Int("port", 8103, "specify the port that should be used for this runner [default: 8103]")
	flag.Parse()

	t := new(runner.Runner)
	defer t.Cleanup()

	err := RegisterProviders(t)
	if err != nil {
		panic(err)
	}

	srv, err := runner.NewRPCServer(&runner.RPCTask{
		T: t,
	})
	if err != nil {
		panic(err)
	}

	tlsConfig, err := ReadConfiguration()
	if err != nil {
		panic(err)
	}

	go srv.ListenAndServeTLS(fmt.Sprintf(":%d", *port), tlsConfig)

	intC := make(chan os.Signal)
	signal.Notify(intC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	fmt.Printf("Received %s signal..\n", <-intC)

	srv.Close()
}

func ReadConfiguration() (tlsConfig *tls.Config, err error) {
	var f *os.File
	f, err = os.Open("config/rpc.json")
	if err != nil {
		return
	}
	dec := json.NewDecoder(f)
	var config runner.RPCConfig
	err = dec.Decode(&config)
	if err != nil {
		return
	}
	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair(config.CrtPath, config.KeyPath)
	if err != nil {
		return
	}
	var b []byte
	b, err = ioutil.ReadFile(config.CAPath)
	if err != nil {
		return
	}

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(b)
	if !ok {
		err = errors.New("Failed to append CA certificates to CertPool")
	}
	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	return
}

func RegisterProviders(r *runner.Runner) (err error) {
	var lp, mp, rp runner.Provider

	lp, err = listener.NewListenerProvider("config/listener.json")
	if err != nil {
		return
	}
	mp, err = mongo.NewMongoProvider("config/mongo.json")
	if err != nil {
		return
	}
	rp, err = remoteexec.NewRemoteExecProvider("config/rpc.json")
	if err != nil {
		return
	}

	r.RegisterProvider(
		lp,
		rp,
		mp,
		ticker.NewTickerProvider(),
		localexec.NewLocalExecProvider(),
	)
	return
}
