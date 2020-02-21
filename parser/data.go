package parser

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

var ErrNoEventName = errors.New("parser: no event name or request id")

const (
	Delimiter = "||"
)

var (
	delimiter = []byte(Delimiter)
)

func DecodeEvent(data []byte) (string, uint64, []byte, error) {
	const noBodyEventParts = 2
	parts := bytes.SplitN(data, delimiter, 3)
	if len(parts) < noBodyEventParts {
		return "", 0, nil, ErrNoEventName
	}
	reqId, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return "", 0, nil, fmt.Errorf("parser: invalid req id: %v", err)
	}
	if len(parts) == noBodyEventParts {
		return string(parts[0]), uint64(reqId), []byte{}, nil
	}
	return string(parts[0]), uint64(reqId), parts[2], nil
}

func EncodeEvent(event string, reqId uint64, body []byte) []byte {
	buf := new(bytes.Buffer)
	EncodeEventToBuffer(buf, event, reqId, body)
	return buf.Bytes()
}

func EncodeEventToBuffer(buf *bytes.Buffer, event string, reqId uint64, body []byte) {
	reqIsStr := strconv.Itoa(int(reqId))
	buf.Grow(len(event) + len(body) + len(delimiter)*2)
	buf.WriteString(event)
	buf.Write(delimiter)
	buf.WriteString(reqIsStr)
	buf.Write(delimiter)
	buf.Write(body)
}
