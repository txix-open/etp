package gen

import (
	"sync/atomic"
)

type ReqIdGenerator interface {
	NewID() uint64
}

type DefaultReqIdGenerator struct {
	nextID uint64
}

func (g *DefaultReqIdGenerator) NewID() uint64 {
	return atomic.AddUint64(&g.nextID, 1)
}
