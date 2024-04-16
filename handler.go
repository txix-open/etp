package etp

import (
	"context"

	"github.com/txix-open/etp/v3/msg"
)

type Handler interface {
	Handle(ctx context.Context, conn *Conn, event msg.Event) []byte
}

type HandlerFunc func(ctx context.Context, conn *Conn, event msg.Event) []byte

func (h HandlerFunc) Handle(ctx context.Context, conn *Conn, event msg.Event) []byte {
	return h(ctx, conn, event)
}

type ConnectHandler func(conn *Conn)

type DisconnectHandler func(conn *Conn, err error)

type ErrorHandler func(conn *Conn, err error)

type mux struct {
	handlers       map[string]Handler
	onUnknownEvent Handler
	onConnect      ConnectHandler
	onDisconnect   DisconnectHandler
	onError        ErrorHandler
}

func newMux() *mux {
	return &mux{
		handlers: make(map[string]Handler),
	}
}

func (m *mux) On(event string, handler Handler) {
	m.handlers[event] = handler
}

func (m *mux) OnConnect(handler ConnectHandler) {
	m.onConnect = handler
}

func (m *mux) OnDisconnect(handler DisconnectHandler) {
	m.onDisconnect = handler
}

func (m *mux) OnError(handler ErrorHandler) {
	m.onError = handler
}

func (m *mux) OnUnknownEvent(handler Handler) {
	m.onUnknownEvent = handler
}

func (m *mux) handle(ctx context.Context, conn *Conn, event msg.Event) []byte {
	handler, ok := m.handlers[event.Name]
	if ok {
		return handler.Handle(ctx, conn, event)
	}
	if m.onUnknownEvent != nil {
		return m.onUnknownEvent.Handle(ctx, conn, event)
	}
	return nil
}

func (m *mux) handleConnect(conn *Conn) {
	if m.onConnect != nil {
		m.onConnect(conn)
	}
}

func (m *mux) handleDisconnect(conn *Conn, err error) {
	if m.onDisconnect != nil {
		m.onDisconnect(conn, err)
	}
}

func (m *mux) handleError(conn *Conn, err error) {
	if m.onError != nil {
		m.onError(conn, err)
	}
}
