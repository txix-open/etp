package internal

import (
	"sync"
)

type Acks struct {
	lock        sync.Locker
	acks        map[uint64]*Ack
	idGenerator *SequenceGenerator
}

func NewAcks() *Acks {
	return &Acks{
		lock:        &sync.Mutex{},
		acks:        make(map[uint64]*Ack),
		idGenerator: NewSequenceGenerator(),
	}
}

func (a *Acks) NextAck() *Ack {
	a.lock.Lock()
	defer a.lock.Unlock()

	id := a.idGenerator.Next()
	ack := NewAck(id)
	a.acks[id] = ack
	return ack
}

func (a *Acks) NotifyAck(ackId uint64, data []byte) {
	a.lock.Lock()
	defer a.lock.Unlock()

	ack, ok := a.acks[ackId]
	if ok {
		ack.Notify(data)
	}
}

func (a *Acks) DeleteAck(ackId uint64) {
	a.lock.Lock()
	defer a.lock.Unlock()

	delete(a.acks, ackId)
}
