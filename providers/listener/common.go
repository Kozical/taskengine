package listener

import (
	"net/http"
)

// ListenerState implements the job.StateObject interface
type ListenerState struct {
	w http.ResponseWriter
	r *http.Request
}

func (l ListenerState) GetProperty(property string) string {
	values := l.r.URL.Query()

	v, ok := values[property]
	if ok {
		return v[0]
	}
	return ""
}
