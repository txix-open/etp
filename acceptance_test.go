package etp_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"github.com/txix-open/etp/v4"
	"github.com/txix-open/etp/v4/msg"
)

type CallHandler struct {
	onConnectCount    atomic.Int32
	onDisconnectCount atomic.Int32
	onErrorCount      atomic.Int32
	events            map[string][][]byte
	lock              sync.Locker
}

func NewCallHandler() *CallHandler {
	return &CallHandler{
		events: make(map[string][][]byte),
		lock:   &sync.Mutex{},
	}
}

func (c *CallHandler) OnConnect(conn *etp.Conn) {
	c.onConnectCount.Add(1)
}

func (c *CallHandler) OnDisconnect(conn *etp.Conn, err error) {
	c.onDisconnectCount.Add(1)
}

func (c *CallHandler) OnError(conn *etp.Conn, err error) {
	c.onErrorCount.Add(1)
}

func (c *CallHandler) Handle(response []byte) etp.Handler {
	return etp.HandlerFunc(func(ctx context.Context, conn *etp.Conn, event msg.Event) []byte {
		c.lock.Lock()
		defer c.lock.Unlock()

		c.events[event.Name] = append(c.events[event.Name], event.Data)
		return response
	})
}

func TestAcceptance(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	srvHandler := NewCallHandler()
	configureSrv := func(srv *etp.Server) {
		srv.OnConnect(srvHandler.OnConnect).
			OnDisconnect(srvHandler.OnDisconnect).
			OnError(srvHandler.OnError).
			On("simpleEvent", srvHandler.Handle([]byte("simpleEventResponse"))).
			On("ackEvent", srvHandler.Handle([]byte("ackEventResponse"))).
			On("emitInHandlerEvent", etp.HandlerFunc(func(ctx context.Context, conn *etp.Conn, event msg.Event) []byte {
				srvHandler.Handle(event.Data).Handle(ctx, conn, event)
				resp, err := conn.EmitWithAck(ctx, "echo", event.Data)
				require.NoError(err)
				require.EqualValues(event.Data, resp)
				return nil
			})).
			OnUnknownEvent(srvHandler.Handle(nil))
	}

	cliHandler := NewCallHandler()
	configureCli := func(cli *etp.Client) {
		cli.OnConnect(cliHandler.OnConnect).
			OnDisconnect(cliHandler.OnDisconnect).
			OnError(cliHandler.OnError).
			On("echo", etp.HandlerFunc(func(ctx context.Context, conn *etp.Conn, event msg.Event) []byte {
				return event.Data
			}))
	}

	cli, _, _ := serve(t, configureSrv, configureCli)

	err := cli.Emit(context.Background(), "simpleEvent", []byte("simpleEventPayload"))
	require.NoError(err)

	resp, err := cli.EmitWithAck(context.Background(), "ackEvent", nil)
	require.NoError(err)
	require.EqualValues([]byte("ackEventResponse"), resp)

	resp, err = cli.EmitWithAck(context.Background(), "emitInHandlerEvent", []byte("emitInHandlerEventPayload"))
	require.NoError(err)
	require.Nil(resp)

	err = cli.Emit(context.Background(), "unknownEvent", []byte("unknownEventPayload"))
	require.NoError(err)

	time.Sleep(1 * time.Second)

	err = cli.Close()
	require.NoError(err)

	time.Sleep(1 * time.Second)

	require.EqualValues(1, srvHandler.onConnectCount.Load())
	require.EqualValues(1, srvHandler.onDisconnectCount.Load())
	require.EqualValues(0, srvHandler.onErrorCount.Load())

	require.Len(srvHandler.events["simpleEvent"], 1)
	require.EqualValues([]byte("simpleEventPayload"), srvHandler.events["simpleEvent"][0])

	require.Len(srvHandler.events["ackEvent"], 1)
	require.Nil(srvHandler.events["ackEvent"][0])

	require.Len(srvHandler.events["emitInHandlerEvent"], 1)
	require.EqualValues([]byte("emitInHandlerEventPayload"), srvHandler.events["emitInHandlerEvent"][0])

	require.Len(srvHandler.events["unknownEvent"], 1)
	require.EqualValues([]byte("unknownEventPayload"), srvHandler.events["unknownEvent"][0])

	require.EqualValues(1, cliHandler.onConnectCount.Load())
	require.EqualValues(1, cliHandler.onDisconnectCount.Load())
	require.EqualValues(0, cliHandler.onErrorCount.Load())
}

func serve(
	t *testing.T,
	srvCfg func(srv *etp.Server),
	cliCfg func(cli *etp.Client),
) (*etp.Client, *etp.Server, *httptest.Server) {
	t.Helper()

	srv := etp.NewServer(etp.WithServerAcceptOptions(
		&websocket.AcceptOptions{
			InsecureSkipVerify: true,
		},
	))
	if srvCfg != nil {
		srvCfg(srv)
	}
	testServer := httptest.NewServer(srv)
	t.Cleanup(func() {
		testServer.Close()
		srv.Shutdown()
	})

	cli := etp.NewClient(etp.WithClientDialOptions(
		&websocket.DialOptions{
			HTTPClient: testServer.Client(),
		},
	))
	if cliCfg != nil {
		cliCfg(cli)
	}
	err := cli.Dial(context.Background(), strings.ReplaceAll(testServer.URL, "http", "ws"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cli.Close()
	})

	return cli, srv, testServer
}
