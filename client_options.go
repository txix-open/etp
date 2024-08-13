package etp

import (
	"github.com/coder/websocket"
)

type DialOptions = websocket.DialOptions

type ClientOption func(*clientOptions)

type clientOptions struct {
	dialOptions *DialOptions
	readLimit   int64
}

func defaultClientOptions() *clientOptions {
	return &clientOptions{
		readLimit: defaultReadLimit,
	}
}

func WithClientDialOptions(opts *DialOptions) ClientOption {
	return func(o *clientOptions) {
		o.dialOptions = opts
	}
}

func WithClientReadLimit(limit int64) ClientOption {
	return func(o *clientOptions) {
		o.readLimit = limit
	}
}
