package etp

import (
	"strconv"
	"sync/atomic"
)

type ConnectionIDGenerator interface {
	NewID() string
}

type defaultIDGenerator struct {
	nextID uint64
}

func (g *defaultIDGenerator) NewID() string {
	id := atomic.AddUint64(&g.nextID, 1)
	return strconv.FormatUint(id, 36)
}
