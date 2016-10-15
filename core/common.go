package core

import (
	"encoding/json"
)

type ParseObject struct {
	Provider   string          `json:"provider"`
	Name       string          `json:"name"`
	Properties json.RawMessage `json:"properties"`
}

type RPCJob struct {
	Name    string
	Objects []ParseObject
}
