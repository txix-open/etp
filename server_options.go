package etp

import (
	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/msg"
)

const (
	// defaultReadLimit is the default maximum message size in bytes.
	defaultReadLimit = 1 * 1024 * 1024
)

// AcceptOptions is an alias for websocket.AcceptOptions.
type AcceptOptions = websocket.AcceptOptions

// ServerOption is a function that configures a serverOptions struct.
type ServerOption func(*serverOptions)

// serverOptions holds configuration options for a Server.
type serverOptions struct {
	acceptOptions *AcceptOptions
	readLimit     int64
	codec         msg.Codec
}

// defaultServerOptions returns the default server configuration.
func defaultServerOptions() *serverOptions {
	return &serverOptions{
		readLimit: defaultReadLimit,
		codec:     msg.NewLineCodec(),
	}
}

// WithServerAcceptOptions sets the WebSocket accept options for the server.
func WithServerAcceptOptions(opts *AcceptOptions) ServerOption {
	return func(options *serverOptions) {
		options.acceptOptions = opts
	}
}

// WithServerReadLimit sets the maximum message size in bytes that the server can read.
func WithServerReadLimit(limit int64) ServerOption {
	return func(options *serverOptions) {
		options.readLimit = limit
	}
}

// WithServerCodec sets the codec for encoding and decoding events.
func WithServerCodec(codec msg.Codec) ServerOption {
	return func(o *serverOptions) {
		o.codec = codec
	}
}
