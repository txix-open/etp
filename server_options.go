package etp

import (
	"nhooyr.io/websocket"
)

const (
	defaultReadLimit = 1 * 1024 * 1024
)

type AcceptOptions = websocket.AcceptOptions

type ServerOption func(*serverOptions)

type serverOptions struct {
	acceptOptions *AcceptOptions
	readLimit     int64
}

func defaultServerOptions() *serverOptions {
	return &serverOptions{
		readLimit: defaultReadLimit,
	}
}

func WithServerAcceptOptions(opts *AcceptOptions) ServerOption {
	return func(options *serverOptions) {
		options.acceptOptions = opts
	}
}

func WithServerReadLimit(limit int64) ServerOption {
	return func(options *serverOptions) {
		options.readLimit = limit
	}
}
