// Package store provides a thread-safe key-value store for arbitrary data.
package store

import (
	"fmt"
	"sync"
)

// Store is a thread-safe key-value store for arbitrary data.
type Store struct {
	data map[string]any
	lock sync.Locker
}

// New creates a new Store instance.
func New() *Store {
	return &Store{
		data: make(map[string]any),
		lock: &sync.Mutex{},
	}
}

// Set stores a value with the specified key.
func (s *Store) Set(key string, value any) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data[key] = value
}

// Get retrieves a value by key. It returns nil if the key does not exist.
func (s *Store) Get(key string) any {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.data[key]
}

// Range iterates over all key-value pairs in the store.
// The iteration stops if the function f returns false.
func (s *Store) Range(f func(key string, value any) bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for k, v := range s.data {
		if !f(k, v) {
			return
		}
	}
}

// Get is a type-safe helper function to retrieve a typed value from the store.
// It returns an error if the key does not exist or if the value cannot be cast to the requested type.
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
