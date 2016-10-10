package main

import (
	"fmt"

	"github.com/Kozical/taskengine/job"
	_ "github.com/Kozical/taskengine/providers"
)

/*
	// <comment>
	<provider> <resource_title> {
		<resource_property>:<resource_value>
	}
*/
func main() {
	p, err := job.NewParser("myjob.job")
	if err != nil {
		panic(err)
	}
	_, err = p.Parse()
	if err != nil {
		panic(err)
	}
	fmt.Scanln()
}
