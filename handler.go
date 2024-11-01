package etp

import (
	"context"
	"sync"

	"github.com/txix-open/etp/v4/msg"
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
	readLock       sync.Locker
	writeLock      sync.Locker
}

func newMux() *mux {
	lock := &sync.RWMutex{}
	return &mux{
		handlers:  make(map[string]Handler),
		readLock:  lock.RLocker(),
		writeLock: lock,
	}
}

func (m *mux) On(event string, handler Handler) {
	m.writeLock.Lock()
	m.handlers[event] = handler
	m.writeLock.Unlock()
}

func (m *mux) OnConnect(handler ConnectHandler) {
	m.writeLock.Lock()
	m.onConnect = handler
	m.writeLock.Unlock()
}

func (m *mux) OnDisconnect(handler DisconnectHandler) {
	m.writeLock.Lock()
	m.onDisconnect = handler
	m.writeLock.Unlock()
}

func (m *mux) OnError(handler ErrorHandler) {
	m.writeLock.Lock()
	m.onError = handler
	m.writeLock.Unlock()
}

func (m *mux) OnUnknownEvent(handler Handler) {
	m.writeLock.Lock()
	m.onUnknownEvent = handler
	m.writeLock.Unlock()
}

func (m *mux) handle(ctx context.Context, conn *Conn, event msg.Event) []byte {
	m.readLock.Lock()
	handler, ok := m.handlers[event.Name]
	if !ok {
		handler = m.onUnknownEvent
	}
	m.readLock.Unlock()

	if handler != nil {
		return handler.Handle(ctx, conn, event)
	}
	return nil
}

func (m *mux) handleConnect(conn *Conn) {
	m.readLock.Lock()
	onConnect := m.onConnect
	m.readLock.Unlock()

	if onConnect != nil {
		onConnect(conn)
	}
}

func (m *mux) handleDisconnect(conn *Conn, err error) {
	m.readLock.Lock()
	onDisconnect := m.onDisconnect
	m.readLock.Unlock()

	if onDisconnect != nil {
		onDisconnect(conn, err)
	}
}

func (m *mux) handleError(conn *Conn, err error) {
	m.readLock.Lock()
	onError := m.onError
	m.readLock.Unlock()

	if onError != nil {
		onError(conn, err)
	}
}
