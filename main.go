package main

import (
	"fmt"

	"github.com/Kozical/taskengine/job"
	"github.com/Kozical/taskengine/providers/listener"
	"github.com/Kozical/taskengine/providers/localexec"
	"github.com/Kozical/taskengine/providers/localpowershell"
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

	RegisterProviders(t)

	err := t.ParseJobs("jobs")
	if err != nil {
		panic(err)
	}

	fmt.Scanln()
}

func RegisterProviders(t *job.TaskEngine) (err error) {
	var lep job.EventProvider
	var meap, rep job.ActionProvider

	lep, err = listener.NewListenerEventProvider("config/listener.json")
	if err != nil {
		return
	}
	meap, err = mongo.NewMongoActionProvider("config/mongo.json")
	if err != nil {
		return
	}
	rep, err = remoteexec.NewRemoteExecActionProvider("config/rpc.json")
	if err != nil {
		return
	}

	t.RegisterEventProvider(
		lep,
		ticker.NewTickerEventProvider(),
	)
	t.RegisterActionProvider(
		rep,
		meap,
		listener.NewListenerActionProvider(),
		localpowershell.NewLocalPowerShellActionProvider(),
		localexec.NewLocalExecActionProvider(),
	)
	return
}
