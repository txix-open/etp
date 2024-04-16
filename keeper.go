package etp

import (
	"context"

	"github.com/txix-open/etp/v3/internal"
	"github.com/txix-open/etp/v3/msg"
)

type keeper struct {
	conn *Conn
	mux  *mux
}

func newKeeper(conn *Conn, mux *mux) *keeper {
	return &keeper{
		conn: conn,
		mux:  mux,
	}
}

func (k *keeper) Serve(ctx context.Context) {
	k.mux.handleConnect(k.conn)
	for {
		err := k.readAndHandleMessage(ctx)
		if err != nil {
			k.mux.handleDisconnect(k.conn, err)
			return
		}
	}
}

func (k *keeper) readAndHandleMessage(ctx context.Context) error {
	_, data, err := k.conn.ws.Read(ctx)
	if err != nil {
		return err
	}

	event, err := msg.UnmarshalEvent(data)
	if err != nil {
		k.mux.onError(k.conn, err)
		return nil
	}

	if internal.IsAckEvent(event.Name) {
		k.conn.notifyAck(event.AckId, event.Data)
		return nil
	}

	go func() {
		response := k.mux.handle(ctx, k.conn, event)
		if !event.IsAckRequired() {
			return
		}

		message := msg.Event{
			Name:  internal.ToAckEvent(event.Name),
			AckId: event.AckId,
			Data:  response,
		}
		err := k.conn.emit(ctx, message)
		if err != nil {
			k.mux.onError(k.conn, err)
		}
	}()

	return nil
}
