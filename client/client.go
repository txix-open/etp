package client

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/integration-system/isp-etp-go/v2/ack"
	"github.com/integration-system/isp-etp-go/v2/bpool"
	"github.com/integration-system/isp-etp-go/v2/gen"
	"github.com/integration-system/isp-etp-go/v2/parser"
	"nhooyr.io/websocket"
)

// TODO: all methods: add check for cl.con != nil? Add lock?

const (
	// TODO
	defaultCloseReason = ""
)

type Client interface {
	Close() error
	CloseWithCode(code websocket.StatusCode, reason string) error
	OnWithAck(event string, f func(data []byte) []byte) Client
	Dial(ctx context.Context, url string) error
	Ping(ctx context.Context) error
	// If registered, all unknown events will be handled here.
	OnDefault(f func(event string, data []byte)) Client
	On(event string, f func(data []byte)) Client
	Unsubscribe(event string) Client
	OnConnect(f func()) Client
	OnDisconnect(f func(error)) Client
	OnError(f func(error)) Client
	Emit(ctx context.Context, event string, body []byte) error
	EmitWithAck(ctx context.Context, event string, body []byte) ([]byte, error)
	Closed() bool
}

type client struct {
	con               *websocket.Conn
	handlers          map[string]func(data []byte)
	ackHandlers       map[string]func(data []byte) []byte
	defaultHandler    func(event string, data []byte)
	connectHandler    func()
	disconnectHandler func(err error)
	errorHandler      func(err error)
	handlersLock      sync.RWMutex
	ackers            *ack.Ackers
	reqIdGenerator    gen.ReqIdGenerator
	globalCtx         context.Context
	cancel            context.CancelFunc
	config            Config
	workersCh         chan eventMsg
	workersWg         sync.WaitGroup
	closeCh           chan struct{}
	closeOnce         sync.Once
	closed            bool
}

type eventMsg struct {
	event string
	reqId uint64
	body  []byte
	buf   *bytes.Buffer
}

func NewClient(config Config) Client {
	if config.WorkersNum <= 0 {
		config.WorkersNum = defaultWorkersNum
	}
	if config.WorkersBufferMultiplier <= 0 {
		config.WorkersBufferMultiplier = defaultWorkersBufferMultiplier
	}
	return &client{
		handlers:       make(map[string]func(data []byte)),
		ackHandlers:    make(map[string]func(data []byte) []byte),
		ackers:         ack.NewAckers(),
		closeCh:        make(chan struct{}),
		workersCh:      make(chan eventMsg, config.WorkersNum*config.WorkersBufferMultiplier),
		reqIdGenerator: &gen.DefaultReqIdGenerator{},
		config:         config,
	}
}

func (cl *client) CloseWithCode(code websocket.StatusCode, reason string) error {
	defer func() {
		cl.close()
	}()
	return cl.con.Close(code, reason)
}

func (cl *client) Close() error {
	return cl.CloseWithCode(websocket.StatusNormalClosure, defaultCloseReason)
}

func (cl *client) Closed() bool {
	return cl.closed
}

func (cl *client) Emit(ctx context.Context, event string, body []byte) error {
	buf := bpool.Get()
	defer bpool.Put(buf)
	parser.EncodeEventToBuffer(buf, event, 0, body)
	return cl.con.Write(ctx, websocket.MessageText, buf.Bytes())
}

func (cl *client) EmitWithAck(ctx context.Context, event string, body []byte) ([]byte, error) {
	reqId := cl.reqIdGenerator.NewID()
	defer cl.ackers.UnregisterAck(reqId)
	buf := bpool.Get()
	defer bpool.Put(buf)

	parser.EncodeEventToBuffer(buf, event, reqId, body)
	acker := cl.ackers.RegisterAck(reqId, ctx, cl.closeCh)
	if err := cl.con.Write(ctx, websocket.MessageText, buf.Bytes()); err != nil {
		return nil, err
	}
	return acker.Await()
}

func (cl *client) Dial(ctx context.Context, url string) error {
	ctx, cancel := context.WithCancel(ctx)
	cl.globalCtx = ctx
	cl.cancel = cancel
	cl.closed = false

	opts := &websocket.DialOptions{
		HTTPClient: cl.config.HttpClient,
		HTTPHeader: cl.config.HttpHeaders,
	}
	c, _, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		return err
	}
	cl.con = c
	if cl.config.ConnectionReadLimit != 0 {
		c.SetReadLimit(cl.config.ConnectionReadLimit)
	}
	for i := 0; i < cl.config.WorkersNum; i++ {
		cl.workersWg.Add(1)
		go cl.worker()
	}

	cl.onConnect()
	go cl.serveRead()
	return nil
}

func (cl *client) Ping(ctx context.Context) error {
	return cl.con.Ping(ctx)
}

func (cl *client) On(event string, f func(data []byte)) Client {
	cl.handlersLock.Lock()
	cl.handlers[event] = f
	cl.handlersLock.Unlock()
	return cl
}

func (cl *client) OnWithAck(event string, f func(data []byte) []byte) Client {
	cl.handlersLock.Lock()
	cl.ackHandlers[event] = f
	cl.handlersLock.Unlock()
	return cl
}

// If registered, all unknown events will be handled here.
func (cl *client) OnDefault(f func(event string, data []byte)) Client {
	cl.handlersLock.Lock()
	cl.defaultHandler = f
	cl.handlersLock.Unlock()
	return cl
}

func (cl *client) Unsubscribe(event string) Client {
	cl.handlersLock.Lock()
	delete(cl.handlers, event)
	cl.handlersLock.Unlock()
	return cl
}

func (cl *client) OnConnect(f func()) Client {
	cl.handlersLock.Lock()
	cl.connectHandler = f
	cl.handlersLock.Unlock()
	return cl
}

func (cl *client) OnDisconnect(f func(error)) Client {
	cl.handlersLock.Lock()
	cl.disconnectHandler = f
	cl.handlersLock.Unlock()
	return cl
}

func (cl *client) OnError(f func(error)) Client {
	cl.handlersLock.Lock()
	cl.errorHandler = f
	cl.handlersLock.Unlock()
	return cl
}

func (cl *client) serveRead() {
	var err error
	for {
		err = cl.readConn()
		if err != nil {
			cl.close()
			cl.onDisconnect(err)
			return
		}
	}
}

func (cl *client) worker() {
	defer cl.workersWg.Done()
	for msg := range cl.workersCh {
		if msg.reqId > 0 {
			if handler, ok := cl.getAckHandler(msg.event); ok {
				answer := handler(msg.body)
				msg.buf.Reset()
				parser.EncodeEventToBuffer(msg.buf, ack.Event(msg.event), msg.reqId, answer)
				err := cl.con.Write(cl.globalCtx, websocket.MessageText, msg.buf.Bytes())
				if err != nil {
					cl.onError(fmt.Errorf("ack to event %s err: %w", msg.event, err))
				}
			}
			bpool.Put(msg.buf)
			continue
		}
		handler, ok := cl.getHandler(msg.event)
		if ok {
			handler(msg.body)
		} else {
			cl.onDefault(msg.event, msg.body)
		}
		bpool.Put(msg.buf)
	}
}

func (cl *client) readConn() error {
	_, r, err := cl.con.Reader(cl.globalCtx)
	if err != nil {
		return err
	}
	needPutBuf := true
	buf := bpool.Get()
	defer func() {
		if needPutBuf {
			bpool.Put(buf)
		}
	}()
	_, err = buf.ReadFrom(r)
	if err != nil {
		return err
	}

	event, reqId, body, err := parser.DecodeEvent(buf.Bytes())
	if err != nil {
		cl.onError(err)
		return nil
	}
	if ack.IsAckEvent(event) {
		if reqId > 0 {
			bodyCopy := make([]byte, len(body))
			copy(bodyCopy, body)
			cl.ackers.TryAck(reqId, bodyCopy)
		}
		return nil
	}
	cl.workersCh <- eventMsg{event: event, reqId: reqId, body: body, buf: buf}
	needPutBuf = false
	return nil
}

func (cl *client) close() {
	cl.closeOnce.Do(func() {
		close(cl.workersCh)
		cl.workersWg.Wait()
		if cl.cancel != nil {
			cl.cancel()
		}
		close(cl.closeCh)
		cl.closed = true
	})
}

func (cl *client) getHandler(event string) (func(data []byte), bool) {
	cl.handlersLock.RLock()
	handler, ok := cl.handlers[event]
	cl.handlersLock.RUnlock()
	return handler, ok
}

func (cl *client) getAckHandler(event string) (func(data []byte) []byte, bool) {
	cl.handlersLock.RLock()
	handler, ok := cl.ackHandlers[event]
	cl.handlersLock.RUnlock()
	return handler, ok
}

func (cl *client) onConnect() {
	cl.handlersLock.RLock()
	handler := cl.connectHandler
	cl.handlersLock.RUnlock()
	if handler != nil {
		handler()
	}
}

func (cl *client) onDisconnect(err error) {
	cl.handlersLock.RLock()
	handler := cl.disconnectHandler
	cl.handlersLock.RUnlock()
	if handler != nil {
		handler(err)
	}
}

func (cl *client) onError(err error) {
	cl.handlersLock.RLock()
	handler := cl.errorHandler
	cl.handlersLock.RUnlock()
	if handler != nil {
		handler(err)
	}
}

func (cl *client) onDefault(event string, data []byte) {
	cl.handlersLock.RLock()
	handler := cl.defaultHandler
	cl.handlersLock.RUnlock()
	if handler != nil {
		handler(event, data)
	}
}
