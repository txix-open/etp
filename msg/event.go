// Package msg provides message types and codecs for event-based communication.
package msg

import (
	"bytes"
)

// Codec defines an interface for encoding and decoding events.
type Codec interface {
	// UnmarshalEvent decodes event data from bytes.
	UnmarshalEvent(data []byte) (Event, error)
	// EncodeEvent encodes an event to bytes.
	EncodeEvent(w *bytes.Buffer, event Event)
}

// Event represents a communication event with a name, optional acknowledgment ID, and data payload.
type Event struct {
	Name  string
	AckId uint64
	Data  []byte
}

// IsAckRequired returns true if the event requires an acknowledgment.
func (e Event) IsAckRequired() bool {
	return e.AckId > 0
}
