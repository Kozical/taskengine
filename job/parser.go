package job

import (
	"bytes"
	"fmt"
)

import (
	"bufio"
	"encoding/json"
	"os"
)

type ParseObject struct {
	Provider   string          `json:"provider"`
	Name       string          `json:"name"`
	Properties json.RawMessage `json:"properties"`
}

func (p *ParseObject) String() string {
	return fmt.Sprintf("provider: %s, name: %s, properties: %q", p.Provider, p.Name, p.Properties)
}

type Parser struct {
	t *Tokenizer
	f *os.File
}

func NewParser(path string) (p *Parser, err error) {
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return
	}
	p = &Parser{
		f: f,
		t: NewTokenizer(bufio.NewReader(f)),
	}
	return
}

func (p *Parser) Parse() (job *Job, err error) {
	var obj []ParseObject

	err = json.Unmarshal(p.toJson(), &obj)
	if err != nil {
		fmt.Printf("Error parsing json data.. %v\n", err)
		return
	}

	job = &Job{
		Name: p.f.Name(),
	}

	RegisteredEventProvidersLock.Lock()
	defer RegisteredEventProvidersLock.Unlock()

	RegisteredActionProvidersLock.Lock()
	defer RegisteredActionProvidersLock.Unlock()

	for i, v := range obj {
		if i == 0 {
			job.Event = &Event{
				Name:       v.Name,
				Properties: JSONPromote(v.Properties),
				Provider:   v.Provider,
				Event:      RegisteredEventProviders[v.Provider],
			}
			continue
		}
		job.Actions = append(job.Actions, &Action{
			Name:       v.Name,
			Properties: JSONPromote(v.Properties),
			Provider:   v.Provider,
			Action:     RegisteredActionProviders[v.Provider],
		})
	}

	job.Register()
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
				buf.WriteString(fmt.Sprintf(`{"provider":"%s",`, JSONEscape(t.val)))
				providerCount++
			case tResourceTitle:
				buf.WriteString(fmt.Sprintf(`"name":"%s",`, JSONEscape(t.val)))
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
				buf.WriteString(fmt.Sprintf(`{"%s":`, JSONEscape(t.val)))
			case tResourcePropertyValue:
				buf.WriteString(fmt.Sprintf(`"%s"`, JSONEscape(t.val)))
				propertyCount++
			case tResourcePropertyArrayValue:
				if propertyArrayCount == 0 {
					buf.WriteByte('[')
				} else {
					buf.WriteByte(',')
				}
				buf.WriteString(fmt.Sprintf(`"%s"`, JSONEscape(t.val)))
				propertyArrayCount++
			case tResourcePropertyMapName:
				if propertyMapCount == 0 {
					buf.WriteByte('{')
				} else {
					buf.WriteByte(',')
				}
				buf.WriteString(fmt.Sprintf(`"%s":`, JSONEscape(t.val)))
			case tResourcePropertyMapValue:
				buf.WriteString(fmt.Sprintf(`"%s"`, JSONEscape(t.val)))
				propertyMapCount++
			}
		}
	}

	if propertyCount > 0 ||
		propertyMapCount > 0 {

		buf.WriteString(`}]`)
		propertyCount = 0
		propertyMapCount = 0
	}
	if propertyArrayCount > 0 {
		buf.WriteByte(']')
		propertyArrayCount = 0
	}
	if providerCount > 0 {
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	fmt.Printf("json: %s\n", buf.String())
	return buf.Bytes()
}

func JSONEscape(data string) string {
	var buf bytes.Buffer
	/*
		\" - double-quote     - 34
		\\ - back-slash       - 92
		\/ - forward-slash    - 47
		\b - backspace        - 8
		\f - form feed        - 12
		\n - line feed        - 10
		\r - carriage return  - 13
		\t - tab              - 9
	*/

	for _, b := range data {
		switch b {
		case 8:
			buf.WriteString(`\b`)
		case 9:
			buf.WriteString(`\t`)
		case 10:
			buf.WriteString(`\n`)
		case 12:
			buf.WriteString(`\f`)
		case 13:
			buf.WriteString(`\r`)
		case 34:
			buf.WriteString(`\"`)
		case 47:
			buf.WriteString(`\/`)
		case 92:
			buf.WriteString(`\\`)
		default:
			buf.WriteRune(b)
		}
	}
	return buf.String()
}

func JSONPromote(data []byte) []byte {
	var insideQuotes bool
	var braceDepth int
	var parts []string
	var lastPart int
	var buf bytes.Buffer
	var d []byte

	if data[0] == '[' && data[len(data)-1] == ']' {
		d = data[1 : len(data)-1]
	} else {
		d = data
	}
	for i, r := range d {
		if r == '"' {
			insideQuotes = !insideQuotes
		}
		if r == '{' {
			braceDepth++
		}
		if r == '}' {
			braceDepth--
		}
		if r == ',' && !insideQuotes && braceDepth == 0 {
			parts = append(parts, string(d[lastPart:i]))
			lastPart = i + 1
		}
	}
	if lastPart > 0 {
		parts = append(parts, string(d[lastPart:len(d)]))
	}
	if lastPart == 0 {
		parts = append(parts, string(d))
	}
	buf.WriteByte('{')
	for i, p := range parts {
		if i > 0 {
			buf.WriteByte(',')
		}
		if p[0] == '{' && p[len(p)-1] == '}' {
			buf.WriteString(p[1 : len(p)-1])
			continue
		}
		buf.WriteString(p)
	}
	buf.WriteByte('}')
	return buf.Bytes()
}