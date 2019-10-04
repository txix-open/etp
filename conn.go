package etp

import (
	"context"
	"github.com/integration-system/isp-etp-go/parser"
	"net/http"
	"net/url"
	"nhooyr.io/websocket"
)

type Conn interface {
	ID() string
	Close() error
	URL() *url.URL
	RemoteAddr() string
	RemoteHeader() http.Header
	Context() interface{}
	SetContext(v interface{})
	Emit(ctx context.Context, event string, body []byte) error
	//EmitWithAck(ctx context.Context, event string, body []byte) (error, []byte)
	read(context.Context) (websocket.MessageType, []byte, error)
}

const (
	// TODO
	defaultCloseReason = ""
)

type conn struct {
	conn       *websocket.Conn
	id         string
	header     http.Header
	remoteAddr string
	url        *url.URL
	context    interface{}
}

func (c *conn) ID() string {
	return c.id
}

func (c *conn) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, defaultCloseReason)
}

func (c *conn) read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return c.conn.Read(ctx)
}

func (c *conn) URL() *url.URL {
	return c.url
}

func (c *conn) RemoteAddr() string {
	return c.remoteAddr
}

func (c *conn) RemoteHeader() http.Header {
	return c.header
}

func (c *conn) Context() interface{} {
	return c.context
}

func (c *conn) SetContext(v interface{}) {
	c.context = v
}

func (c *conn) Emit(ctx context.Context, event string, body []byte) error {
	data := parser.EncodeBody(event, body)
	return c.conn.Write(ctx, websocket.MessageText, data)
}
