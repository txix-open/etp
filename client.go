package etp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/coder/websocket"
	"github.com/txix-open/etp/v3/internal"
)

var (
	ErrClientClosed = errors.New("client closed")
)

type Client struct {
	mux         *mux
	idGenerator *internal.IdGenerator
	opts        *clientOptions
	conn        *Conn
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
		conn:        nil,
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

	if c.conn != nil {
		return errors.New("already connected")
	}

	ws, resp, err := websocket.Dial(ctx, url, c.opts.dialOptions)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	ws.SetReadLimit(c.opts.readLimit)

	id := c.idGenerator.Next()
	conn := newConn(id, resp.Request, ws)
	c.conn = conn

	keeper := newKeeper(conn, c.mux)
	go func() {
		defer func() {
			_ = ws.CloseNow()
		}()

		keeper.Serve(context.Background())

		c.lock.Lock()
		defer c.lock.Unlock()
		if c.conn != nil && c.conn.Id() == id {
			c.conn = nil
		}
	}()
	return nil
}

func (c *Client) Emit(ctx context.Context, event string, data []byte) error {
	if c.conn == nil {
		return ErrClientClosed
	}
	return c.conn.Emit(ctx, event, data)
}

func (c *Client) EmitWithAck(ctx context.Context, event string, data []byte) ([]byte, error) {
	if c.conn == nil {
		return nil, ErrClientClosed
	}
	return c.conn.EmitWithAck(ctx, event, data)
}

func (c *Client) Ping(ctx context.Context) error {
	if c.conn == nil {
		return ErrClientClosed
	}
	return c.conn.Ping(ctx)
}

func (c *Client) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.conn == nil {
		return ErrClientClosed
	}

	err := c.conn.Close()
	c.conn = nil
	return err
}
