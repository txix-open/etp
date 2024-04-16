package etp

import (
	"nhooyr.io/websocket"
)

const (
	defaultReadLimit = 1 * 1024 * 1024
)

type ServerOption func(*ServerOptions)

type ServerOptions struct {
	acceptOptions *websocket.AcceptOptions
	readLimit     int64
}

func DefaultServerOptions() *ServerOptions {
	return &ServerOptions{
		readLimit: defaultReadLimit,
	}
}

func WithServerAcceptOptions(opts *websocket.AcceptOptions) ServerOption {
	return func(options *ServerOptions) {
		options.acceptOptions = opts
	}
}

func WithServerReadLimit(limit int64) ServerOption {
	return func(options *ServerOptions) {
		options.readLimit = limit
	}
}
