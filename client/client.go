package client

import (
	"context"
	"github.com/integration-system/isp-etp-go/parser"
	"nhooyr.io/websocket"
	"sync"
)

const (
	// TODO
	defaultCloseReason = ""
)

type Client interface {
	Close() error
	//OnWithAck(event string, f func(data []byte) string) Client
	Dial(ctx context.Context, url string) error
	On(event string, f func(data []byte)) Client
	Unsubscribe(event string) Client
	OnConnect(f func()) Client
	OnDisconnect(f func(error)) Client
	OnError(f func(error)) Client
	Emit(ctx context.Context, event string, body []byte) error
}

type client struct {
	con               *websocket.Conn
	handlers          map[string]func(data []byte)
	connectHandler    func()
	disconnectHandler func(err error)
	errorHandler      func(err error)
	handlersLock      sync.RWMutex
	globalCtx         context.Context
	cancel            context.CancelFunc
	config            Config
}

func NewClient(config Config) Client {
	return &client{
		handlers: map[string]func(data []byte){},
		config:   config,
	}
}

func (cl *client) Close() error {
	defer func() {
		if cl.cancel != nil {
			cl.cancel()
		}
	}()
	return cl.con.Close(websocket.StatusNormalClosure, defaultCloseReason)

}

func (cl *client) Emit(ctx context.Context, event string, body []byte) error {
	data := parser.EncodeBody(event, body)
	return cl.con.Write(ctx, websocket.MessageText, data)
}

func (cl *client) Dial(ctx context.Context, url string) error {
	ctx, cancel := context.WithCancel(ctx)
	cl.globalCtx = ctx
	cl.cancel = cancel

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
	cl.onConnect()
	go cl.serveRead()
	return nil
}

func (cl *client) serveRead() {
	defer func() {
		err := cl.Close()
		cl.onDisconnect(err)
	}()

	for {
		_, bytes, err := cl.con.Read(cl.globalCtx)
		if err != nil {
			cl.onError(err)
			return
		}
		event, body, err := parser.ParseData(bytes)
		if err != nil {
			cl.onError(err)
			continue
		}

		handler, ok := cl.getHandler(event)
		if ok {
			handler(body)
		}
	}
}

func (cl *client) getHandler(event string) (func(data []byte), bool) {
	cl.handlersLock.RLock()
	handler, ok := cl.handlers[event]
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

func (cl *client) On(event string, f func(data []byte)) Client {
	cl.handlersLock.Lock()
	cl.handlers[event] = f
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
