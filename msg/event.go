package msg

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	Delimiter        = "||"
	noBodyEventParts = 2
)

var (
	delimiter = []byte(Delimiter)
)

type Event struct {
	Name  string
	AckId uint64
	Data  []byte
}

func (e Event) IsAckRequired() bool {
	return e.AckId > 0
}

func UnmarshalEvent(data []byte) (Event, error) {
	parts := bytes.SplitN(data, delimiter, 3)
	if len(parts) < noBodyEventParts {
		return Event{}, fmt.Errorf("expected format: <eventName>||<ackId>||<eventData>, got: %s", string(data))
	}

	ackId, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return Event{}, fmt.Errorf("parse ackId: %w", err)
	}

	var eventData []byte
	if len(parts) > noBodyEventParts {
		eventData = parts[2]
	}

	return Event{
		Name:  string(parts[0]),
		AckId: uint64(ackId),
		Data:  eventData,
	}, nil
}

func MarshalEvent(event Event) []byte {
	buff := bytes.NewBuffer(nil)
	EncodeEvent(buff, event)
	return buff.Bytes()
}

func EncodeEvent(buff *bytes.Buffer, event Event) {
	ackId := strconv.FormatInt(int64(event.AckId), 10)
	buff.Grow(len(event.Name) + len(ackId) + len(event.Data) + len(delimiter)*2)
	buff.WriteString(event.Name)
	buff.Write(delimiter)
	buff.WriteString(ackId)
	if len(event.Data) > 0 {
		buff.Write(delimiter)
		buff.Write(event.Data)
	}
}
