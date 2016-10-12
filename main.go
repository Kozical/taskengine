package main

import (
	"fmt"

	"github.com/Kozical/taskengine/job"
	"github.com/Kozical/taskengine/providers/listener"
	"github.com/Kozical/taskengine/providers/localexec"
	"github.com/Kozical/taskengine/providers/mongo"
	"github.com/Kozical/taskengine/providers/remoteexec"
	"github.com/Kozical/taskengine/providers/ticker"
)

/*
	// <comment>
	<provider> <resource_title> {
		<resource_property>:<resource_value>
	}
*/
func main() {
	t := job.NewTaskEngine()
	defer t.Cleanup()

	err := RegisterProviders(t)
	if err != nil {
		panic(err)
	}

	err = t.ParseJobs("jobs")
	if err != nil {
		panic(err)
	}

	fmt.Scanln()
}

func RegisterProviders(t *job.TaskEngine) (err error) {
	var lp, mp, rp job.Provider

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

	t.RegisterProvider(
		lp,
		rp,
		mp,
		ticker.NewTickerProvider(),
		localexec.NewLocalExecProvider(),
	)
	return
}
