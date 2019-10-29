package gen

import (
	"strconv"
	"sync/atomic"
)

type ConnectionIDGenerator interface {
	NewID() string
}

type DefaultIDGenerator struct {
	nextID uint64
}

func (g *DefaultIDGenerator) NewID() string {
	id := atomic.AddUint64(&g.nextID, 1)
	return strconv.FormatUint(id, 36)
}
