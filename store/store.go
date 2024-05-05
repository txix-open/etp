package store

import (
	"fmt"
	"sync"
)

type Store struct {
	data map[string]any
	lock sync.Locker
}

func New() *Store {
	return &Store{
		data: make(map[string]any),
		lock: &sync.Mutex{},
	}
}

func (s *Store) Set(key string, value any) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data[key] = value
}

func (s *Store) Range(f func(key string, value any) bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for k, v := range s.data {
		if !f(k, v) {
			return
		}
	}
}

func Get[T any](store *Store, key string) (T, error) {
	store.lock.Lock()
	defer store.lock.Unlock()

	var empty T

	value, ok := store.data[key]
	if !ok {
		return empty, fmt.Errorf("value for key %s not found", key)
	}

	casted, ok := value.(T)
	if !ok {
		return empty, fmt.Errorf("expected type %T, but got %T", empty, value)
	}

	return casted, nil
}
