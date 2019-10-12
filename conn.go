package etp

import (
	"context"
	"github.com/integration-system/isp-etp-go/ack"
	"github.com/integration-system/isp-etp-go/gen"
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
	EmitWithAck(ctx context.Context, event string, body []byte) ([]byte, error)
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
}

func (c *conn) ID() string {
	return c.id
}

func (c *conn) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, defaultCloseReason)
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

	acker := c.ackers.RegisterAck(reqId, ctx, c.closeCh)
	if err := c.conn.Write(ctx, websocket.MessageText, body); err != nil {
		return nil, err
	}

	return acker.Await()
}

func (c *conn) tryAck(reqId uint64, data []byte) bool {
	return c.ackers.TryAck(reqId, data)
}

func (c *conn) read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return c.conn.Read(ctx)
}
