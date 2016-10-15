package core

import (
	"bytes"
)

// input:  [ {"a":"value"},{"b":"value"} ]
// result: { "a":"value", "b":"value" }
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
