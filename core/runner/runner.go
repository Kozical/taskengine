package runner

import (
	"fmt"
	"reflect"
	"strings"
)

type Runner struct {
	providers []Provider
}

func NewRunner() (r *Runner) {
	r = new(Runner)
	return
}

func (r *Runner) RegisterProviders(providers ...Provider) {
	for _, p := range providers {
		r.providers = append(r.providers, p)
	}
}

func (r *Runner) GetProvider(name string) Provider {
	for _, p := range r.providers {
		if strings.HasSuffix(strings.ToLower(reflect.TypeOf(p).String()), fmt.Sprintf("%s%s", strings.ToLower(name), "provider")) {
			return p
		}
	}
	return nil
}
