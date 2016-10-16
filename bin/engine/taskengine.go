package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Kozical/taskengine/core/engine"
)

/*
	// <comment>
	<provider> <resource_title> {
		<resource_property>:<resource_value>
	}
*/
func main() {
	tlsConfig, err := ReadConfiguration()
	if err != nil {
		panic(err)
	}

	endpoints := []string{"self.lab.local:8103"}

	mgr := engine.NewRPCMgr(15, endpoints, tlsConfig)
	defer mgr.Cleanup()

	jobs, err := engine.ParseJobsInDirectory("jobs")
	if err != nil {
		panic(err)
	}

	//Implement Job queuing and job assignment
	//after clients become available
	//Started using go-routine to connect to runners
	//will need to implement better 'watching' routine
	//so that we can identify when a 'dead' node comes back
	//online and throw him back into the rotation
	err = mgr.DispatchJobs(jobs)
	if err != nil {
		panic(err)
	}

	//t.AssignRunners()
	//t.DispatchJobs()

	fmt.Scanln()
}

func ReadConfiguration() (tlsConfig *tls.Config, err error) {
	var f *os.File
	f, err = os.Open("config/rpc.json")
	if err != nil {
		return
	}
	dec := json.NewDecoder(f)
	var config engine.RPCConfig
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
		RootCAs:      pool,
	}
	return
}
