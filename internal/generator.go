package internal

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
)

type SequenceGenerator struct {
	next *atomic.Uint64
}

func NewSequenceGenerator() *SequenceGenerator {
	return &SequenceGenerator{
		next: &atomic.Uint64{},
	}
}

func (g *SequenceGenerator) Next() uint64 {
	return g.next.Add(1)
}

type IdGenerator struct {
}

func NewIdGenerator() *IdGenerator {
	return &IdGenerator{}
}

func (g *IdGenerator) Next() string {
	value := make([]byte, 16)
	_, err := rand.Read(value)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(value)
}
