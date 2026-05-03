package etp

import (
	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/msg"
)

const (
	defaultReadLimit = 1 * 1024 * 1024
)

type AcceptOptions = websocket.AcceptOptions

type ServerOption func(*serverOptions)

type serverOptions struct {
	acceptOptions *AcceptOptions
	readLimit     int64
	codec         msg.Codec
}

func defaultServerOptions() *serverOptions {
	return &serverOptions{
		readLimit: defaultReadLimit,
		codec:     msg.NewLineCodec(),
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

func WithServerCodec(codec msg.Codec) ServerOption {
	return func(o *serverOptions) {
		o.codec = codec
	}
}
