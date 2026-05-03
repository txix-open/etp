package etp

import (
	"github.com/coder/websocket"
	"github.com/txix-open/etp/v4/msg"
)

type DialOptions = websocket.DialOptions

type ClientOption func(*clientOptions)

type clientOptions struct {
	dialOptions *DialOptions
	readLimit   int64
	codec       msg.Codec
}

func defaultClientOptions() *clientOptions {
	return &clientOptions{
		readLimit: defaultReadLimit,
		codec:     msg.NewLineCodec(),
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

func WithClientCodec(codec msg.Codec) ClientOption {
	return func(o *clientOptions) {
		o.codec = codec
	}
}
