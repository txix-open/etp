package etp

import (
	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/msg"
)

// DialOptions is an alias for websocket.DialOptions.
type DialOptions = websocket.DialOptions

// ClientOption is a function that configures a clientOptions struct.
type ClientOption func(*clientOptions)

// clientOptions holds configuration options for a Client.
type clientOptions struct {
	dialOptions *DialOptions
	readLimit   int64
	codec       msg.Codec
}

// defaultClientOptions returns the default client configuration.
func defaultClientOptions() *clientOptions {
	return &clientOptions{
		readLimit: defaultReadLimit,
		codec:     msg.NewLineCodec(),
	}
}

// WithClientDialOptions sets the WebSocket dial options for the client.
func WithClientDialOptions(opts *DialOptions) ClientOption {
	return func(o *clientOptions) {
		o.dialOptions = opts
	}
}

// WithClientReadLimit sets the maximum message size in bytes that the client can read.
func WithClientReadLimit(limit int64) ClientOption {
	return func(o *clientOptions) {
		o.readLimit = limit
	}
}

// WithClientCodec sets the codec for encoding and decoding events.
func WithClientCodec(codec msg.Codec) ClientOption {
	return func(o *clientOptions) {
		o.codec = codec
	}
}
