package etp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/bpool"
	"github.com/txix-open/etp/v4/internal"
	"github.com/txix-open/etp/v4/msg"
	"github.com/txix-open/etp/v4/store"
)

type Conn struct {
	id      string
	request *http.Request
	ws      *websocket.Conn
	data    *store.Store
	acks    *internal.Acks
}

func newConn(
	id string,
	request *http.Request,
	ws *websocket.Conn,
) *Conn {
	return &Conn{
		id:      id,
		request: request,
		ws:      ws,
		data:    store.New(),
		acks:    internal.NewAcks(),
	}
}

func (c *Conn) Id() string {
	return c.id
}

func (c *Conn) HttpRequest() *http.Request {
	return c.request
}

func (c *Conn) Data() *store.Store {
	return c.data
}

func (c *Conn) Emit(ctx context.Context, event string, data []byte) error {
	message := msg.Event{
		Name:  event,
		AckId: 0,
		Data:  data,
	}
	return c.emit(ctx, message)
}

func (c *Conn) EmitWithAck(ctx context.Context, event string, data []byte) ([]byte, error) {
	ack := c.acks.NextAck()
	defer c.acks.DeleteAck(ack.Id())

	message := msg.Event{
		Name:  event,
		AckId: ack.Id(),
		Data:  data,
	}
	err := c.emit(ctx, message)
	if err != nil {
		return nil, err
	}

	response, err := ack.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait ack: %w", err)
	}

	return response, nil
}

func (c *Conn) Ping(ctx context.Context) error {
	return c.ws.Ping(ctx)
}

func (c *Conn) Close() error {
	return c.ws.Close(websocket.StatusNormalClosure, "")
}

func (c *Conn) emit(ctx context.Context, event msg.Event) error {
	buff := bpool.Get()
	defer bpool.Put(buff)

	msg.EncodeEvent(buff, event)

	err := c.ws.Write(ctx, websocket.MessageText, buff.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

func (c *Conn) notifyAck(ackId uint64, data []byte) {
	c.acks.NotifyAck(ackId, data)
}
