package etp

import (
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/internal"
)

// Server represents a WebSocket server for event-based communication.
// It is safe for concurrent use.
type Server struct {
	idGenerator *internal.IdGenerator
	mux         *mux
	rooms       *Rooms
	opts        *serverOptions
}

// NewServer creates a new Server instance with the provided options.
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

// On registers an event handler for the specified event name.
// It returns the Server to allow method chaining.
func (s *Server) On(event string, handler Handler) *Server {
	s.mux.On(event, handler)
	return s
}

// OnConnect registers a handler to be called when a client connects.
// It returns the Server to allow method chaining.
func (s *Server) OnConnect(handler ConnectHandler) *Server {
	s.mux.OnConnect(handler)
	return s
}

// OnDisconnect registers a handler to be called when a client disconnects.
// It returns the Server to allow method chaining.
func (s *Server) OnDisconnect(handler DisconnectHandler) *Server {
	s.mux.OnDisconnect(handler)
	return s
}

// OnError registers a handler to be called when an error occurs.
// It returns the Server to allow method chaining.
func (s *Server) OnError(handler ErrorHandler) *Server {
	s.mux.OnError(handler)
	return s
}

// OnUnknownEvent registers a handler to be called when an unknown event is received.
// It returns the Server to allow method chaining.
func (s *Server) OnUnknownEvent(handler Handler) *Server {
	s.mux.OnUnknownEvent(handler)
	return s
}

// Rooms returns the Rooms instance for managing connection groups.
func (s *Server) Rooms() *Rooms {
	return s.rooms
}

// ServeHTTP handles HTTP requests by upgrading them to WebSocket connections.
// It implements the http.Handler interface.
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
	conn := newConn(id, r, ws, s.opts.codec)

	s.rooms.add(conn)
	defer s.rooms.remove(conn)

	keeper := newKeeper(conn, s.mux)
	keeper.Serve(r.Context())
}

// Shutdown closes all active connections on the server.
func (s *Server) Shutdown() {
	for _, conn := range s.rooms.AllConns() {
		_ = conn.Close()
	}
}
