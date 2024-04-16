package etp

import (
	"nhooyr.io/websocket"
)

type ClientOption func(*ClientOptions)

type ClientOptions struct {
	dialOptions *websocket.DialOptions
	readLimit   int64
}

func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		readLimit: defaultReadLimit,
	}
}

func WithClientDialOptions(opts *websocket.DialOptions) ClientOption {
	return func(o *ClientOptions) {
		o.dialOptions = opts
	}
}

func WithClientReadLimit(limit int64) ClientOption {
	return func(o *ClientOptions) {
		o.readLimit = limit
	}
}
