package etp

import (
	"context"
	"errors"
	"github.com/hashicorp/go-multierror"
	"github.com/integration-system/isp-etp-go/parser"
	"io"
	"net/http"
	"nhooyr.io/websocket"
	"sync"
)

type Server interface {
	Close()
	ServeHttp(w http.ResponseWriter, r *http.Request)
	//OnWithAck(event string, f func(conn Conn, data []byte) string) Server
	OnDefault(f func(event string, conn Conn, data []byte)) Server
	On(event string, f func(conn Conn, data []byte)) Server
	Unsubscribe(event string) Server
	OnConnect(f func(Conn)) Server
	OnDisconnect(f func(Conn, error)) Server
	OnError(f func(Conn, error)) Server

	Rooms() RoomStore
	BroadcastToRoom(room string, event string, data []byte) error
	BroadcastToAll(event string, data []byte) error
}

type server struct {
	handlers          map[string]func(conn Conn, data []byte)
	rooms             RoomStore
	defaultHandler    func(event string, conn Conn, data []byte)
	connectHandler    func(conn Conn)
	disconnectHandler func(conn Conn, err error)
	errorHandler      func(conn Conn, err error)
	idGenerator       ConnectionIDGenerator
	handlersLock      sync.RWMutex
	globalCtx         context.Context
	cancel            context.CancelFunc
	config            ServerConfig
}

func NewServer(ctx context.Context, config ServerConfig) Server {
	ctx, cancel := context.WithCancel(ctx)
	return &server{
		handlers:    map[string]func(conn Conn, data []byte){},
		rooms:       NewRoomStore(),
		globalCtx:   ctx,
		cancel:      cancel,
		idGenerator: &defaultIDGenerator{},
		config:      config,
	}
}

func (s *server) Close() {
	s.cancel()
}

func (s *server) ServeHttp(w http.ResponseWriter, r *http.Request) {
	if s.config.RequestHandler != nil {
		err := s.config.RequestHandler(r)
		if err != nil {
			s.onError(nil, err)
			return
		}
	}
	options := websocket.AcceptOptions{
		InsecureSkipVerify: s.config.InsecureSkipVerify,
	}
	c, err := websocket.Accept(w, r, &options)
	if err != nil {
		s.onError(nil, err)
		return
	}
	if s.config.ConnectionReadLimit != 0 {
		c.SetReadLimit(s.config.ConnectionReadLimit)
	}
	id := s.idGenerator.NewID()
	connect := &conn{
		conn:       c,
		id:         id,
		header:     r.Header,
		remoteAddr: r.RemoteAddr,
		url:        r.URL,
	}
	s.rooms.Add(connect)
	s.onConnect(connect)
	go s.serveRead(connect)
}

func (s *server) serveRead(con Conn) {
	defer func() {
		err := con.Close()
		s.rooms.Remove(con)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
		}
		s.onDisconnect(con, err)
	}()

	for {
		_, bytes, err := con.read(s.globalCtx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			s.onError(con, err)
			return
		}
		event, body, err := parser.ParseData(bytes)
		if err != nil {
			s.onError(con, err)
			continue
		}

		handler, ok := s.getHandler(event)
		if ok {
			handler(con, body)
		} else {
			s.onDefault(event, con, body)
		}
	}
}

func (s *server) getHandler(event string) (func(conn Conn, data []byte), bool) {
	s.handlersLock.RLock()
	handler, ok := s.handlers[event]
	s.handlersLock.RUnlock()
	return handler, ok
}

func (s *server) onConnect(conn Conn) {
	s.handlersLock.RLock()
	handler := s.connectHandler
	s.handlersLock.RUnlock()
	if handler != nil {
		handler(conn)
	}
}

func (s *server) onDisconnect(conn Conn, err error) {
	s.handlersLock.RLock()
	handler := s.disconnectHandler
	s.handlersLock.RUnlock()
	if handler != nil {
		handler(conn, err)
	}
}

func (s *server) onError(conn Conn, err error) {
	s.handlersLock.RLock()
	handler := s.errorHandler
	s.handlersLock.RUnlock()
	if handler != nil {
		handler(conn, err)
	}
}

func (s *server) onDefault(event string, conn Conn, data []byte) {
	s.handlersLock.RLock()
	handler := s.defaultHandler
	s.handlersLock.RUnlock()
	if handler != nil {
		handler(event, conn, data)
	}
}

func (s *server) Rooms() RoomStore {
	return s.rooms
}

func (s *server) On(event string, f func(conn Conn, data []byte)) Server {
	s.handlersLock.Lock()
	s.handlers[event] = f
	s.handlersLock.Unlock()
	return s
}

// If registered, all unknown events will be handled here.
func (s *server) OnDefault(f func(event string, conn Conn, data []byte)) Server {
	s.handlersLock.Lock()
	s.defaultHandler = f
	s.handlersLock.Unlock()
	return s
}

func (s *server) Unsubscribe(event string) Server {
	s.handlersLock.Lock()
	delete(s.handlers, event)
	s.handlersLock.Unlock()
	return s
}

func (s *server) OnConnect(f func(Conn)) Server {
	s.handlersLock.Lock()
	s.connectHandler = f
	s.handlersLock.Unlock()
	return s
}

func (s *server) OnDisconnect(f func(Conn, error)) Server {
	s.handlersLock.Lock()
	s.disconnectHandler = f
	s.handlersLock.Unlock()
	return s
}

// Conn may be nil if error occurs on connection upgrade or in RequestHandler
func (s *server) OnError(f func(Conn, error)) Server {
	s.handlersLock.Lock()
	s.errorHandler = f
	s.handlersLock.Unlock()
	return s
}

// Returns go-multierror
func (s *server) BroadcastToRoom(room string, event string, data []byte) error {
	var errs error
	conns := s.rooms.ToBroadcast(room)
	for _, conn := range conns {
		err := conn.Emit(s.globalCtx, event, data)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// Returns go-multierror
func (s *server) BroadcastToAll(event string, data []byte) error {
	var errs error
	rooms := s.rooms.Rooms()
	conns := s.rooms.ToBroadcast(rooms...)
	for _, conn := range conns {
		err := conn.Emit(s.globalCtx, event, data)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}
