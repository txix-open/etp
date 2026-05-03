// Package etp provides a client-server communication library using WebSockets.
// It supports event-based messaging, acknowledgments, and room-based broadcasting.
package etp

import (
	"context"
	"errors"
	"fmt"

	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/internal"
)

var (
	// ErrClientClosed is returned when the client is closed or not connected.
	ErrClientClosed = errors.New("client closed")
)

// Client represents a WebSocket client for event-based communication.
// It is safe for concurrent use.
type Client struct {
	mux         *mux
	idGenerator *internal.IdGenerator
	opts        *clientOptions
	conn        *atomic.Pointer[Conn]
	lock        sync.Locker
}

// NewClient creates a new Client instance with the provided options.
func NewClient(opts ...ClientOption) *Client {
	options := defaultClientOptions()
	for _, opt := range opts {
		opt(options)
	}
	return &Client{
		mux:         newMux(),
		idGenerator: internal.NewIdGenerator(),
		opts:        options,
		conn:        &atomic.Pointer[Conn]{},
		lock:        &sync.Mutex{},
	}
}

// On registers an event handler for the specified event name.
// It returns the Client to allow method chaining.
func (c *Client) On(event string, handler Handler) *Client {
	c.mux.On(event, handler)
	return c
}

// OnConnect registers a handler to be called when the client connects.
// It returns the Client to allow method chaining.
func (c *Client) OnConnect(handler ConnectHandler) *Client {
	c.mux.OnConnect(handler)
	return c
}

// OnDisconnect registers a handler to be called when the client disconnects.
// It returns the Client to allow method chaining.
func (c *Client) OnDisconnect(handler DisconnectHandler) *Client {
	c.mux.OnDisconnect(handler)
	return c
}

// OnError registers a handler to be called when an error occurs.
// It returns the Client to allow method chaining.
func (c *Client) OnError(handler ErrorHandler) *Client {
	c.mux.OnError(handler)
	return c
}

// OnUnknownEvent registers a handler to be called when an unknown event is received.
// It returns the Client to allow method chaining.
func (c *Client) OnUnknownEvent(handler Handler) *Client {
	c.mux.OnUnknownEvent(handler)
	return c
}

// Dial establishes a WebSocket connection to the specified URL.
// It returns an error if already connected or if the WebSocket dial fails.
func (c *Client) Dial(ctx context.Context, url string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.conn.Load() != nil {
		return errors.New("already connected")
	}

	ws, resp, err := websocket.Dial(ctx, url, c.opts.dialOptions)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	ws.SetReadLimit(c.opts.readLimit)

	id := c.idGenerator.Next()
	conn := newConn(id, resp.Request, ws, c.opts.codec)
	c.conn.Store(conn)

	keeper := newKeeper(conn, c.mux)
	go func() {
		defer func() {
			_ = ws.CloseNow()
		}()

		keeper.Serve(context.Background())

		c.lock.Lock()
		defer c.lock.Unlock()
		conn := c.conn.Load()
		if conn != nil && conn.Id() == id {
			c.conn.Store(nil)
		}
	}()
	return nil
}

// Emit sends an event with the specified data to the server.
// It returns ErrClientClosed if the client is not connected.
func (c *Client) Emit(ctx context.Context, event string, data []byte) error {
	conn := c.conn.Load()
	if conn == nil {
		return ErrClientClosed
	}
	return conn.Emit(ctx, event, data)
}

// EmitWithAck sends an event with the specified data to the server and waits for an acknowledgment.
// It returns ErrClientClosed if the client is not connected.
// The response data from the server is returned on success.
func (c *Client) EmitWithAck(ctx context.Context, event string, data []byte) ([]byte, error) {
	conn := c.conn.Load()
	if conn == nil {
		return nil, ErrClientClosed
	}
	return conn.EmitWithAck(ctx, event, data)
}

// Ping sends a ping to the server to check the connection health.
// It returns ErrClientClosed if the client is not connected.
func (c *Client) Ping(ctx context.Context) error {
	conn := c.conn.Load()
	if conn == nil {
		return ErrClientClosed
	}
	return conn.Ping(ctx)
}

// Close closes the client connection.
// It returns ErrClientClosed if the client is already closed.
func (c *Client) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	conn := c.conn.Load()
	if conn == nil {
		return ErrClientClosed
	}

	err := conn.Close()
	c.conn.Store(nil)
	return err
}
