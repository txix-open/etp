package etp

import (
	"fmt"
	"net/http"

	"github.com/txix-open/etp/v3/internal"
	"nhooyr.io/websocket"
)

type Server struct {
	idGenerator *internal.IdGenerator
	mux         *mux
	rooms       *Rooms
	opts        *ServerOptions
}

func NewServer(opts ...ServerOption) *Server {
	options := DefaultServerOptions()
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

	s.rooms.Add(conn)
	defer s.rooms.Remove(conn)

	keeper := newKeeper(conn, s.mux)
	keeper.Serve(r.Context())
}
