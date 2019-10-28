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
	parts := bytes.SplitN(data, delimiter, 3)
	if len(parts) < 2 {
		return "", 0, nil, ErrNoEventName
	}
	reqId, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return "", 0, nil, fmt.Errorf("parser: invalid req id: %v", err)
	}
	if len(parts) == 2 {
		return string(parts[0]), uint64(reqId), []byte{}, nil
	}
	return string(parts[0]), uint64(reqId), parts[2], nil
}

func EncodeEvent(event string, reqId uint64, body []byte) []byte {
	var buf bytes.Buffer
	reqIsStr := strconv.Itoa(int(reqId))
	buf.Grow(len(event) + len(body) + len(delimiter)*2)
	buf.WriteString(event)
	buf.Write(delimiter)
	buf.WriteString(reqIsStr)
	buf.Write(delimiter)
	buf.Write(body)
	return buf.Bytes()
}
