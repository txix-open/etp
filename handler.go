package etp

import (
	"context"
	"sync"

	"github.com/txix-open/etp/v4/msg"
)

// Handler is an interface for handling events.
// The Handle method processes an event and returns a response.
type Handler interface {
	Handle(ctx context.Context, conn *Conn, event msg.Event) []byte
}

// HandlerFunc is a function type that implements the Handler interface.
type HandlerFunc func(ctx context.Context, conn *Conn, event msg.Event) []byte

// Handle calls the underlying function.
func (h HandlerFunc) Handle(ctx context.Context, conn *Conn, event msg.Event) []byte {
	return h(ctx, conn, event)
}

// ConnectHandler is a function type called when a connection is established.
type ConnectHandler func(conn *Conn)

// DisconnectHandler is a function type called when a connection is closed.
type DisconnectHandler func(conn *Conn, err error)

// ErrorHandler is a function type called when an error occurs.
type ErrorHandler func(conn *Conn, err error)

// mux is an internal event multiplexer that manages handlers.
type mux struct {
	handlers       map[string]Handler
	onUnknownEvent Handler
	onConnect      ConnectHandler
	onDisconnect   DisconnectHandler
	onError        ErrorHandler
	readLock       sync.Locker
	writeLock      sync.Locker
}

// newMux creates a new mux instance.
func newMux() *mux {
	lock := &sync.RWMutex{}
	return &mux{
		handlers:  make(map[string]Handler),
		readLock:  lock.RLocker(),
		writeLock: lock,
	}
}

// On registers an event handler for the specified event name.
func (m *mux) On(event string, handler Handler) {
	m.writeLock.Lock()
	m.handlers[event] = handler
	m.writeLock.Unlock()
}

// OnConnect registers a handler to be called when a connection is established.
func (m *mux) OnConnect(handler ConnectHandler) {
	m.writeLock.Lock()
	m.onConnect = handler
	m.writeLock.Unlock()
}

// OnDisconnect registers a handler to be called when a connection is closed.
func (m *mux) OnDisconnect(handler DisconnectHandler) {
	m.writeLock.Lock()
	m.onDisconnect = handler
	m.writeLock.Unlock()
}

// OnError registers a handler to be called when an error occurs.
func (m *mux) OnError(handler ErrorHandler) {
	m.writeLock.Lock()
	m.onError = handler
	m.writeLock.Unlock()
}

// OnUnknownEvent registers a handler to be called when an unknown event is received.
func (m *mux) OnUnknownEvent(handler Handler) {
	m.writeLock.Lock()
	m.onUnknownEvent = handler
	m.writeLock.Unlock()
}

// handle dispatches an event to the appropriate handler.
// It returns the response from the handler.
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

// handleConnect calls the registered connect handler.
func (m *mux) handleConnect(conn *Conn) {
	m.readLock.Lock()
	onConnect := m.onConnect
	m.readLock.Unlock()

	if onConnect != nil {
		onConnect(conn)
	}
}

// handleDisconnect calls the registered disconnect handler.
func (m *mux) handleDisconnect(conn *Conn, err error) {
	m.readLock.Lock()
	onDisconnect := m.onDisconnect
	m.readLock.Unlock()

	if onDisconnect != nil {
		onDisconnect(conn, err)
	}
}

// handleError calls the registered error handler.
func (m *mux) handleError(conn *Conn, err error) {
	m.readLock.Lock()
	onError := m.onError
	m.readLock.Unlock()

	if onError != nil {
		onError(conn, err)
	}
}
