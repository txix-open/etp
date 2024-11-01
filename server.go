package etp

import (
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/internal"
)

type Server struct {
	idGenerator *internal.IdGenerator
	mux         *mux
	rooms       *Rooms
	opts        *serverOptions
}

func NewServer(opts ...ServerOption) *Server {
	options := defaultServerOptions()
	for _, opt := range opts {
		opt(options)
	}
	return &Server{
		idGenerator: internal.NewIdGenerator(),
		mux:         newMux(),
		rooms:       newRooms(),
		opts:        options,
	}
}

func (s *Server) On(event string, handler Handler) *Server {
	s.mux.On(event, handler)
	return s
}

func (s *Server) OnConnect(handler ConnectHandler) *Server {
	s.mux.OnConnect(handler)
	return s
}

func (s *Server) OnDisconnect(handler DisconnectHandler) *Server {
	s.mux.OnDisconnect(handler)
	return s
}

func (s *Server) OnError(handler ErrorHandler) *Server {
	s.mux.OnError(handler)
	return s
}

func (s *Server) OnUnknownEvent(handler Handler) *Server {
	s.mux.OnUnknownEvent(handler)
	return s
}

func (s *Server) Rooms() *Rooms {
	return s.rooms
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Accept(w, r, s.opts.acceptOptions)
	if err != nil {
		s.mux.handleError(nil, fmt.Errorf("websocket accept error: %w", err))
		return
	}
	defer func() {
		_ = ws.CloseNow()
	}()

	ws.SetReadLimit(s.opts.readLimit)

	id := s.idGenerator.Next()
	conn := newConn(id, r, ws)

	s.rooms.add(conn)
	defer s.rooms.remove(conn)

	keeper := newKeeper(conn, s.mux)
	keeper.Serve(r.Context())
}

func (s *Server) Shutdown() {
	for _, conn := range s.rooms.AllConns() {
		_ = conn.Close()
	}
}
