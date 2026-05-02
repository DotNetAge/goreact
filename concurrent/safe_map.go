// Package concurrent provides thread-safe generic data structures.
package concurrent

import "sync"

// SafeMap is a generic thread-safe map wrapper.
// It replaces the common pattern of mutex + map[K]V with a single type.
type SafeMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// NewSafeMap creates a new empty SafeMap.
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		m: make(map[K]V),
	}
}

// NewSafeMapWithCapacity creates a new SafeMap with pre-allocated capacity.
func NewSafeMapWithCapacity[K comparable, V any](cap int) *SafeMap[K, V] {
	return &SafeMap[K, V]{
		m: make(map[K]V, cap),
	}
}

// Get retrieves a value by key. Returns zero value and false if not found.
func (s *SafeMap[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	return v, ok
}

// Set sets a key-value pair.
func (s *SafeMap[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

// Delete removes a key from the map.
func (s *SafeMap[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

// Len returns the number of entries.
func (s *SafeMap[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// Range calls fn for each entry. Stops early if fn returns false.
func (s *SafeMap[K, V]) Range(fn func(key K, value V) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.m {
		if !fn(k, v) {
			break
		}
	}
}

// Keys returns all keys in the map.
func (s *SafeMap[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]K, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values in the map.
func (s *SafeMap[K, V]) Values() []V {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vals := make([]V, 0, len(s.m))
	for _, v := range s.m {
		vals = append(vals, v)
	}
	return vals
}

// Clear removes all entries.
func (s *SafeMap[K, V]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.m)
}

// PutIfAbsent sets key to value only if key does not already exist.
// Returns the existing value (if any) and whether the new value was inserted.
func (s *SafeMap[K, V]) PutIfAbsent(key K, value V) (existing V, inserted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.m[key]; ok {
		return v, false
	}
	s.m[key] = value
	return value, true
}

// SafeValue is a generic thread-safe wrapper around a single value.
type SafeValue[T any] struct {
	mu sync.RWMutex
	v  T
}

// NewSafeValue creates a new SafeValue with the given initial value.
func NewSafeValue[T any](initial T) *SafeValue[T] {
	return &SafeValue[T]{v: initial}
}

// Get returns the current value.
func (s *SafeValue[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.v
}

// Set sets the value.
func (s *SafeValue[T]) Set(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v = v
}

// Update atomically applies fn to the current value.
func (s *SafeValue[T]) Update(fn func(T) T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v = fn(s.v)
}
