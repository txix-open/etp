package etp

import (
	"context"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/internal"
	"sync"
	"sync/atomic"
)

var (
	ErrClientClosed = errors.New("client closed")
)

type Client struct {
	mux         *mux
	idGenerator *internal.IdGenerator
	opts        *clientOptions
	conn        *atomic.Pointer[Conn]
	lock        sync.Locker
}

func NewClient(opts ...ClientOption) *Client {
	options := defaultClientOptions()
	for _, opt := range opts {
		opt(options)
	}
	return &Client{
		mux:         newMux(),
		idGenerator: internal.NewIdGenerator(),
		opts:        options,
		lock:        &sync.Mutex{},
		conn:        &atomic.Pointer[Conn]{},
	}
}

func (c *Client) On(event string, handler Handler) *Client {
	c.mux.On(event, handler)
	return c
}

func (c *Client) OnConnect(handler ConnectHandler) *Client {
	c.mux.OnConnect(handler)
	return c
}

func (c *Client) OnDisconnect(handler DisconnectHandler) *Client {
	c.mux.OnDisconnect(handler)
	return c
}

func (c *Client) OnError(handler ErrorHandler) *Client {
	c.mux.OnError(handler)
	return c
}

func (c *Client) OnUnknownEvent(handler Handler) *Client {
	c.mux.OnUnknownEvent(handler)
	return c
}

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
	conn := newConn(id, resp.Request, ws)
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

func (c *Client) Emit(ctx context.Context, event string, data []byte) error {
	conn := c.conn.Load()
	if conn == nil {
		return ErrClientClosed
	}
	return conn.Emit(ctx, event, data)
}

func (c *Client) EmitWithAck(ctx context.Context, event string, data []byte) ([]byte, error) {
	conn := c.conn.Load()
	if conn == nil {
		return nil, ErrClientClosed
	}
	return conn.EmitWithAck(ctx, event, data)
}

func (c *Client) Ping(ctx context.Context) error {
	conn := c.conn.Load()
	if conn == nil {
		return ErrClientClosed
	}
	return conn.Ping(ctx)
}

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
