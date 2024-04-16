package bpool

import (
	"bytes"
	"sync"
)

var (
	bpool = sync.Pool{New: func() any {
		return bytes.NewBuffer(make([]byte, 1024))
	}}
)

func Get() *bytes.Buffer {
	b := bpool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func Put(b *bytes.Buffer) {
	bpool.Put(b)
}
