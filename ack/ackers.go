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
	m       map[uint64]*Acker
	connCtx context.Context
}

func (a *Ackers) RegisterAck(reqCtx context.Context, reqId uint64) *Acker {
	a.Lock()
	acker := NewAcker(a.connCtx, reqCtx)
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
	acker, ok := a.m[reqId]
	a.Unlock()
	if ok {
		acker.Notify(data)
		return true
	}
	return false
}

func NewAckers(connCtx context.Context) *Ackers {
	return &Ackers{
		Mutex:   sync.Mutex{},
		m:       make(map[uint64]*Acker),
		connCtx: connCtx,
	}
}

func Event(event string) string {
	return ackPrefix + event
}

func IsAckEvent(event string) bool {
	return strings.HasPrefix(event, ackPrefix)
}
