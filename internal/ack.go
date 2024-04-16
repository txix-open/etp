package internal

import (
	"context"
	"strings"
)

type Ack struct {
	id       uint64
	response chan []byte
}

func NewAck(id uint64) *Ack {
	return &Ack{
		id:       id,
		response: make(chan []byte, 1),
	}
}

func (a *Ack) Wait(ctx context.Context) ([]byte, error) {
	select {
	case data := <-a.response:
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (a *Ack) Notify(data []byte) {
	select {
	case a.response <- data:
	default:
	}
}

func (a *Ack) Id() uint64 {
	return a.id
}

const (
	ackPrefix = "__ack_"
)

func ToAckEvent(event string) string {
	return ackPrefix + event
}

func IsAckEvent(event string) bool {
	return strings.HasPrefix(event, ackPrefix)
}
