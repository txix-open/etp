package internal

import (
	"sync/atomic"
)

type IdGenerator struct {
	next *atomic.Uint64
}

func NewIdGenerator() *IdGenerator {
	return &IdGenerator{
		next: &atomic.Uint64{},
	}
}

func (g *IdGenerator) Next() uint64 {
	return g.next.Add(1)
}
