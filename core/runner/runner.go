package runner

import (
	"crypto/tls"
)

type Runner struct {
	providers []Provider
	Jobs      []*Job
}

func NewRunner(poolsize int, endpoints []string, tlsConfig *tls.Config) (r *Runner) {
	r = new(Runner)
	return
}

func (r *Runner) Cleanup() {
	/*
		Cleanup no longer happens here
		Perhaps we can use a new RPCTask
		Function to dispatch a shutdown
		but do we want the runners to
		stop because we are?
	*/
	for _, j := range r.Jobs {
		for _, t := range j.Tasks {
			t.Provider.Cleanup()
		}
	}
}

func (r *Runner) RegisterProvider(providers ...Provider) {
	for _, p := range providers {
		r.providers = append(r.providers, p)
	}
}

func (r *Runner) GetProvider(name string) Provider {
	for _, p := range r.providers {
		if p.Name() == name {
			return p
		}
	}
	return nil
}
