package ack

import (
	"context"
	"errors"
)

var (
	ErrRequestTimeout = errors.New("ack: timeout")
	ErrConnClose      = errors.New("ack: connection closed")
)

type Acker struct {
	reqCtx      context.Context
	connCtx     context.Context
	ackDataChan chan []byte
}

func (a *Acker) Await() ([]byte, error) {
	select {
	case data := <-a.ackDataChan:
		return data, nil
	case <-a.reqCtx.Done():
		return nil, ErrRequestTimeout
	case <-a.connCtx.Done():
		return nil, ErrConnClose
	}
}

func (a *Acker) Notify(data []byte) {
	select {
	case a.ackDataChan <- data:
	case <-a.reqCtx.Done():
		return
	case <-a.connCtx.Done():
		return
	}
}

func NewAcker(connCtx, reqCtx context.Context) *Acker {
	return &Acker{
		connCtx:     connCtx,
		reqCtx:      reqCtx,
		ackDataChan: make(chan []byte),
	}
}
