package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Kozical/taskengine/core/runner"

	"github.com/Kozical/taskengine/providers/listener"
	"github.com/Kozical/taskengine/providers/localexec"
	"github.com/Kozical/taskengine/providers/mongo"
	"github.com/Kozical/taskengine/providers/ticker"
)

func main() {
	logPath := flag.String("logpath", "", "specify a directory for log output, if not specified logs will be written to Stdout")
	port := flag.Int("port", 8103, "specify the port that should be used for this runner [default: 8103]")
	listenerPath := flag.String("listener", "config/listener.json", "specify the path to the listener config [default: config/listener.json]")
	mongoPath := flag.String("mongo", "config/mongo.json", "specify the path to the mongo config [default: config/mongo.json]")

	flag.Parse()

	ConfigureLogging(*logPath)

	t := new(runner.Runner)

	err := RegisterProviders(t, *mongoPath, *listenerPath)
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

	log.Printf("Received %s signal..\n", <-intC)

	srv.Close()
}

func ConfigureLogging(logPath string) {
	if logPath == "" {
		return
	}
	currentFile := fmt.Sprintf("%s-%s.log", os.Args[0], time.Now().Format("01-02-2006"))
	path := filepath.Join(logPath, currentFile)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE, os.ModeAppend)
	if err != nil {
		panic(err)
	}

	log.SetOutput(f)
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

func RegisterProviders(r *runner.Runner, mongoPath, listenerPath string) (err error) {
	var lp, mp runner.Provider

	if _, err = os.Stat(listenerPath); err == nil {
		lp, err = listener.NewListenerProvider("config/listener.json")
		if err != nil {
			return
		}
		r.RegisterProviders(lp)
	}

	if _, err = os.Stat(mongoPath); err == nil {
		mp, err = mongo.NewMongoProvider(mongoPath)
		if err != nil {
			return
		}
		r.RegisterProviders(mp)
	}

	r.RegisterProviders(
		ticker.NewTickerProvider(),
		localexec.NewLocalExecProvider(),
	)
	return
}
