package parser

import (
	"bytes"
	"errors"
)

var ErrNoEventName = errors.New("parser: no event name in data")

const Delimiter = `||`

// Returns event, body
func ParseData(data []byte) (string, []byte, error) {
	counter := 0
	indexLast := 0
	for i, b := range data {
		if string(b) == "|" {
			counter++
		} else {
			counter = 0
		}
		if counter == 2 {
			indexLast = i
			break
		}
	}
	if indexLast == 0 {
		return "", nil, ErrNoEventName
	}
	return string(data[:indexLast-1]), data[indexLast+1:], nil
}

// Returns data
func EncodeBody(event string, body []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(event) + len(body) + len(Delimiter))
	buf.WriteString(event)
	buf.WriteString(Delimiter)
	buf.Write(body)
	return buf.Bytes()
}
