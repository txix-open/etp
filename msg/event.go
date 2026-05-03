package msg

import (
	"bytes"
)

type Codec interface {
	UnmarshalEvent(data []byte) (Event, error)
	EncodeEvent(w *bytes.Buffer, event Event)
}
type Event struct {
	Name  string
	AckId uint64
	Data  []byte
}

func (e Event) IsAckRequired() bool {
	return e.AckId > 0
}
