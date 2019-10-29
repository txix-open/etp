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
	ackDataChan chan []byte
	closeCh     chan struct{}
}

func (a *Acker) Await() ([]byte, error) {
	select {
	case data := <-a.ackDataChan:
		return data, nil
	case <-a.reqCtx.Done():
		return nil, ErrRequestTimeout
	case <-a.closeCh:
		return nil, ErrConnClose
	}
}

func (a *Acker) Notify(data []byte) {
	select {
	case a.ackDataChan <- data:
	default:

	}
}

func NewAcker(reqCtx context.Context, closeCh chan struct{}) *Acker {
	return &Acker{
		closeCh:     closeCh,
		reqCtx:      reqCtx,
		ackDataChan: make(chan []byte),
	}
}
