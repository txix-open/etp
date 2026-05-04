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

// Conn represents a WebSocket connection associated with a client or server.
type Conn struct {
	id      string
	request *http.Request
	ws      *websocket.Conn
	data    *store.Store
	acks    *internal.Acks
	codec   msg.Codec
}

// newConn creates a new connection with the provided parameters.
func newConn(
	id string,
	request *http.Request,
	ws *websocket.Conn,
	codec msg.Codec,
) *Conn {
	return &Conn{
		id:      id,
		request: request,
		ws:      ws,
		data:    store.New(),
		acks:    internal.NewAcks(),
		codec:   codec,
	}
}

// Id returns the unique identifier for the connection.
func (c *Conn) Id() string {
	return c.id
}

// HttpRequest returns the underlying HTTP request associated with the connection.
func (c *Conn) HttpRequest() *http.Request {
	return c.request
}

// Data returns the key-value store for connection-specific data.
func (c *Conn) Data() *store.Store {
	return c.data
}

// Emit sends an event with the specified data over the connection.
func (c *Conn) Emit(ctx context.Context, event string, data []byte) error {
	message := msg.Event{
		Name:  event,
		AckId: 0,
		Data:  data,
	}
	return c.emit(ctx, message)
}

// EmitWithAck sends an event with the specified data and waits for an acknowledgment.
// It returns the response data from the receiver on success.
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

// Ping sends a WebSocket ping to check connection health.
func (c *Conn) Ping(ctx context.Context) error {
	return c.ws.Ping(ctx)
}

// Close closes the WebSocket connection with a normal closure status.
func (c *Conn) Close() error {
	return c.ws.Close(websocket.StatusNormalClosure, "")
}

// emit encodes and writes an event to the WebSocket connection.
func (c *Conn) emit(ctx context.Context, event msg.Event) error {
	buff := bpool.Get()
	defer bpool.Put(buff)

	err := c.codec.EncodeEvent(buff, event)
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	err = c.ws.Write(ctx, websocket.MessageText, buff.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// notifyAck notifies the acknowledgment handler with the received response data.
func (c *Conn) notifyAck(ackId uint64, data []byte) {
	c.acks.NotifyAck(ackId, data)
}
