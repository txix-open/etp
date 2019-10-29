package ack

import (
	"context"
	"strings"
	"sync"
)

const (
	ackPrefix = "__ack_"
)

type Ackers struct {
	sync.Mutex
	m map[uint64]*Acker
}

func (a *Ackers) RegisterAck(reqId uint64, reqCtx context.Context, closeCh chan struct{}) *Acker {
	a.Lock()
	acker := NewAcker(reqCtx, closeCh)
	a.m[reqId] = acker
	a.Unlock()
	return acker
}

func (a *Ackers) UnregisterAck(reqId uint64) {
	a.Lock()
	delete(a.m, reqId)
	a.Unlock()
}

func (a *Ackers) TryAck(reqId uint64, data []byte) bool {
	a.Lock()
	defer a.Unlock()

	if acker, ok := a.m[reqId]; ok {
		acker.Notify(data)
		return true
	}
	return false
}

func NewAckers() *Ackers {
	return &Ackers{
		Mutex: sync.Mutex{},
		m:     make(map[uint64]*Acker),
	}
}

func Event(event string) string {
	return ackPrefix + event
}

func IsAckEvent(event string) bool {
	return strings.HasPrefix(event, ackPrefix)
}
