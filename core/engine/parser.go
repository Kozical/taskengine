package engine

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Kozical/taskengine/core"
)

type Parser struct {
	t *Tokenizer
	f *os.File
}

func ParseJobsInDirectory(path string) (jobs []*core.RPCJob, err error) {
	var jobPath string

	if filepath.IsAbs(path) == false {
		jobPath = path
	} else {
		jobPath, err = filepath.Abs(path)
		if err != nil {
			return
		}
	}

	var files []string
	files, err = filepath.Glob(filepath.Join(jobPath, "*.job"))

	for _, f := range files {
		var fd *os.File
		var job *core.RPCJob

		if fd, err = os.Open(f); err != nil {
			return
		}
		p := &Parser{
			f: fd,
			t: NewTokenizer(bufio.NewReader(fd)),
		}
		job, err = p.parse()
		if err != nil {
			return
		}
		log.Printf("loaded job: %s\n", job.Name)
		jobs = append(jobs, job)
	}
	return
}

func (p *Parser) parse() (job *core.RPCJob, err error) {
	var obj []core.ParseObject

	err = json.Unmarshal(p.toJson(), &obj)
	if err != nil {
		err = fmt.Errorf("Error parsing json data.. %v\n", err)
		return
	}
	job = &core.RPCJob{
		Name:    filepath.Base(p.f.Name()),
		Objects: obj,
	}
	return
}

func (p *Parser) toJson() []byte {
	go p.t.Tokenize()

	var buf bytes.Buffer

	var providerCount int
	var propertyCount int
	var propertyArrayCount int
	var propertyMapCount int

	buf.WriteByte('[')
Loop:
	for {
		select {
		case t, ok := <-p.t.C:
			if !ok {
				break Loop
			}
			switch t.typ {
			case tProvider:
				if propertyCount > 0 {
					buf.WriteString(`}]`)
					propertyCount = 0
				}
				if propertyMapCount > 0 {
					buf.WriteString(`}}]`)
					propertyMapCount = 0
				}
				if propertyArrayCount > 0 {
					buf.WriteString(`]}]`)
					propertyArrayCount = 0
				}
				if providerCount > 0 {
					buf.WriteString(`},`)
				}
				buf.WriteString(fmt.Sprintf(`{"provider":"%s",`, core.JSONEscape(t.val)))
				providerCount++
			case tResourceTitle:
				buf.WriteString(fmt.Sprintf(`"name":"%s",`, core.JSONEscape(t.val)))
			case tResourcePropertyName:
				if propertyCount == 0 &&
					propertyArrayCount == 0 &&
					propertyMapCount == 0 {
					buf.WriteString(`"properties":[`)
				}
				if propertyCount > 0 {
					buf.WriteString(`},`)
					propertyCount = 0
				}
				if propertyArrayCount > 0 {
					buf.WriteString(`]},`)
					propertyArrayCount = 0
				}
				if propertyMapCount > 0 {
					buf.WriteString(`}},`)
					propertyMapCount = 0
				}
				buf.WriteString(fmt.Sprintf(`{"%s":`, core.JSONEscape(t.val)))
			case tResourcePropertyValue:
				buf.WriteString(fmt.Sprintf(`"%s"`, core.JSONEscape(t.val)))
				propertyCount++
			case tResourcePropertyArrayValue:
				if propertyArrayCount == 0 {
					buf.WriteByte('[')
				} else {
					buf.WriteByte(',')
				}
				buf.WriteString(fmt.Sprintf(`"%s"`, core.JSONEscape(t.val)))
				propertyArrayCount++
			case tResourcePropertyMapName:
				if propertyMapCount == 0 {
					buf.WriteByte('{')
				} else {
					buf.WriteByte(',')
				}
				buf.WriteString(fmt.Sprintf(`"%s":`, core.JSONEscape(t.val)))
			case tResourcePropertyMapValue:
				buf.WriteString(fmt.Sprintf(`"%s"`, core.JSONEscape(t.val)))
				propertyMapCount++
			}
		}
	}

	if propertyCount > 0 {
		buf.WriteString(`}]`)
		propertyCount = 0
	}
	if propertyMapCount > 0 {
		buf.WriteString(`}}]`)
		propertyMapCount = 0
	}
	if propertyArrayCount > 0 {
		buf.WriteString(`]}]`)
		propertyArrayCount = 0
	}
	if providerCount > 0 {
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.Bytes()
}
