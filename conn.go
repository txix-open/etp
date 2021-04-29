package etp

import (
	"context"
	"net/http"
	"net/url"

	"github.com/integration-system/isp-etp-go/v2/ack"
	"github.com/integration-system/isp-etp-go/v2/bpool"
	"github.com/integration-system/isp-etp-go/v2/gen"
	"github.com/integration-system/isp-etp-go/v2/parser"
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
	Ping(ctx context.Context) error
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
	connCtx    context.Context
	cancelCtx  func()
}

func (c *conn) ID() string {
	return c.id
}

func (c *conn) Close() error {
	defer c.close()
	return c.conn.Close(websocket.StatusNormalClosure, defaultCloseReason)
}

func (c *conn) Closed() bool {
	return c.connCtx.Err() != nil
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

func (c *conn) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

func (c *conn) Emit(ctx context.Context, event string, body []byte) error {
	buf := bpool.Get()
	defer bpool.Put(buf)
	parser.EncodeEventToBuffer(buf, event, 0, body)
	return c.conn.Write(ctx, websocket.MessageText, buf.Bytes())
}

func (c *conn) EmitWithAck(ctx context.Context, event string, body []byte) ([]byte, error) {
	reqId := c.gen.NewID()
	defer c.ackers.UnregisterAck(reqId)
	buf := bpool.Get()
	defer bpool.Put(buf)

	parser.EncodeEventToBuffer(buf, event, reqId, body)
	acker := c.ackers.RegisterAck(ctx, reqId)
	if err := c.conn.Write(ctx, websocket.MessageText, buf.Bytes()); err != nil {
		return nil, err
	}

	return acker.Await()
}

func (c *conn) tryAck(reqId uint64, data []byte) bool {
	return c.ackers.TryAck(reqId, data)
}

func (c *conn) close() {
	c.cancelCtx()
}
