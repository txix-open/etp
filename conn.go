package etp

import (
	"context"
	"github.com/integration-system/isp-etp-go/ack"
	"github.com/integration-system/isp-etp-go/gen"
	"github.com/integration-system/isp-etp-go/parser"
	"net/http"
	"net/url"
	"nhooyr.io/websocket"
	"sync"
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
	EmitWithAck(ctx context.Context, event string, body []byte) ([]byte, error)
	Closed() bool
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
	gen        gen.ReqIdGenerator
	ackers     *ack.Ackers
	closeCh    chan struct{}
	closeOnce  sync.Once
	closed     bool
}

func (c *conn) ID() string {
	return c.id
}

func (c *conn) Close() error {
	c.closeAckers()
	return c.conn.Close(websocket.StatusNormalClosure, defaultCloseReason)
}

func (c *conn) Closed() bool {
	return c.closed
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
	data := parser.EncodeEvent(event, 0, body)
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *conn) EmitWithAck(ctx context.Context, event string, body []byte) ([]byte, error) {
	reqId := c.gen.NewID()
	defer c.ackers.UnregisterAck(reqId)
	data := parser.EncodeEvent(event, reqId, body)

	acker := c.ackers.RegisterAck(reqId, ctx, c.closeCh)
	if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return nil, err
	}

	return acker.Await()
}

func (c *conn) tryAck(reqId uint64, data []byte) bool {
	return c.ackers.TryAck(reqId, data)
}

func (c *conn) closeAckers() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		c.closed = true
	})
}

func (c *conn) read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return c.conn.Read(ctx)
}
