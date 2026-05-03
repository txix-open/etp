package msg

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	// Delimiter is the separator used between event name, ack ID, and data.
	Delimiter        = "||"
	noBodyEventParts = 2
)

var (
	delimiter = []byte(Delimiter)
)

// LineCodec is a simple text-based codec that uses "||" as a delimiter.
// The format is: <eventName>||<ackId>||<eventData>
type LineCodec struct{}

// NewLineCodec creates a new LineCodec instance.
func NewLineCodec() LineCodec {
	return LineCodec{}
}

// UnmarshalEvent decodes an event from the line-based format.
// It returns an error if the format is invalid or if the ackId cannot be parsed.
func (LineCodec) UnmarshalEvent(data []byte) (Event, error) {
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

// EncodeEvent encodes an event to the line-based format.
func (LineCodec) EncodeEvent(buff *bytes.Buffer, event Event) {
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
