package etp

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/integration-system/isp-etp-go/v2/ack"
	"github.com/integration-system/isp-etp-go/v2/bpool"
	"github.com/integration-system/isp-etp-go/v2/gen"
	"github.com/integration-system/isp-etp-go/v2/parser"
	"nhooyr.io/websocket"
)

type Server interface {
	Close()
	ServeHttp(w http.ResponseWriter, r *http.Request)
	OnWithAck(event string, f func(conn Conn, data []byte) []byte) Server
	// If registered, all unknown events will be handled here.
	OnDefault(f func(event string, conn Conn, data []byte)) Server
	On(event string, f func(conn Conn, data []byte)) Server
	Unsubscribe(event string) Server
	OnConnect(f func(Conn)) Server
	OnDisconnect(f func(Conn, error)) Server
	// Conn may be nil if error occurs on connection upgrade or in RequestHandler.
	OnError(f func(Conn, error)) Server

	Rooms() RoomStore
	BroadcastToRoom(room string, event string, data []byte) error
	BroadcastToAll(event string, data []byte) error
}

type server struct {
	handlers          map[string]func(conn Conn, data []byte)
	ackHandlers       map[string]func(conn Conn, data []byte) []byte
	rooms             RoomStore
	defaultHandler    func(event string, conn Conn, data []byte)
	connectHandler    func(conn Conn)
	disconnectHandler func(conn Conn, err error)
	errorHandler      func(conn Conn, err error)
	idGenerator       gen.ConnectionIDGenerator
	reqIdGenerator    gen.ReqIdGenerator
	handlersLock      sync.RWMutex
	closed            int64
	config            ServerConfig
}

func NewServer(_ context.Context, config ServerConfig) Server {
	return &server{
		handlers:       make(map[string]func(conn Conn, data []byte)),
		ackHandlers:    make(map[string]func(conn Conn, data []byte) []byte),
		rooms:          NewRoomStore(),
		idGenerator:    &gen.DefaultIDGenerator{},
		reqIdGenerator: &gen.DefaultReqIdGenerator{},
		config:         config,
	}
}

func (s *server) Close() {
	atomic.StoreInt64(&s.closed, 1)
	conns := s.rooms.AllConns()
	for _, connection := range conns {
		connection.(*conn).close()
	}
}

func (s *server) ServeHttp(w http.ResponseWriter, r *http.Request) {
	if s.config.RequestHandler != nil {
		err := s.config.RequestHandler(r)
		if err != nil {
			s.onError(nil, err)
			return
		}
	}

	if atomic.LoadInt64(&s.closed) == 1 {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable)+": websocket server is closed", http.StatusServiceUnavailable)
		return
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
	connCtx, cancelCtx := context.WithCancel(r.Context())
	id := s.idGenerator.NewID()
	connect := &conn{
		conn:       c,
		id:         id,
		header:     r.Header,
		remoteAddr: r.RemoteAddr,
		url:        r.URL,
		ackers:     ack.NewAckers(connCtx),
		gen:        s.reqIdGenerator,
		connCtx:    connCtx,
		cancelCtx:  cancelCtx,
	}
	s.rooms.Add(connect)
	s.onConnect(connect)
	s.serveRead(connect)
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

func (s *server) OnWithAck(event string, f func(conn Conn, data []byte) []byte) Server {
	s.handlersLock.Lock()
	s.ackHandlers[event] = f
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

// Conn may be nil if error occurs on connection upgrade or in RequestHandler.
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
		err := conn.Emit(context.Background(), event, data)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// Returns go-multierror
func (s *server) BroadcastToAll(event string, data []byte) error {
	var errs error
	conns := s.rooms.AllConns()
	for _, conn := range conns {
		err := conn.Emit(context.Background(), event, data)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

func (s *server) serveRead(con *conn) {
	var err error
	for {
		err = s.readConn(con)
		if err != nil {
			s.rooms.Remove(con)
			con.close()
			s.onDisconnect(con, err)
			return
		}
	}
}

func (s *server) readConn(con *conn) error {
	_, r, err := con.conn.Reader(con.connCtx)
	if err != nil {
		return err
	}
	buf := bpool.Get()
	defer bpool.Put(buf)
	_, err = buf.ReadFrom(r)
	if err != nil {
		return err
	}

	event, reqId, body, err := parser.DecodeEvent(buf.Bytes())
	if err != nil {
		s.onError(con, err)
		return nil
	}

	if ack.IsAckEvent(event) {
		if reqId > 0 {
			bodyCopy := make([]byte, len(body))
			copy(bodyCopy, body)
			con.tryAck(reqId, bodyCopy)
		}
		return nil
	}
	if reqId > 0 {
		if handler, ok := s.getAckHandler(event); ok {
			answer := handler(con, body)
			buf.Reset()
			parser.EncodeEventToBuffer(buf, ack.Event(event), reqId, answer)
			err := con.conn.Write(con.connCtx, websocket.MessageText, buf.Bytes())
			if err != nil {
				s.onError(con, fmt.Errorf("ack to event %s err: %w", event, err))
			}
		}
		return nil
	}
	if handler, ok := s.getHandler(event); ok {
		handler(con, body)
	} else {
		s.onDefault(event, con, body)
	}
	return nil
}

func (s *server) getHandler(event string) (func(conn Conn, data []byte), bool) {
	s.handlersLock.RLock()
	handler, ok := s.handlers[event]
	s.handlersLock.RUnlock()
	return handler, ok
}

func (s *server) getAckHandler(event string) (func(conn Conn, data []byte) []byte, bool) {
	s.handlersLock.RLock()
	handler, ok := s.ackHandlers[event]
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
