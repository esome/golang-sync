// Package syncutils provides thread-safe data structures and other utilities for working with concurrency.

package syncutils

import (
	"iter"
	"maps"
	"sync"
)

// NewMap creates a new thread-safe map of type K, V
// An empty map is allocated with enough space to hold the
// specified number of elements. The size may be omitted, in which case
// a small starting size is allocated.
func NewMap[K comparable, V any](size ...int) *Map[K, V] {
	if len(size) > 0 {
		return &Map[K, V]{syncMap[K, V]{items: make(map[K]V, size[0])}}
	}

	return &Map[K, V]{syncMap[K, V]{items: make(map[K]V)}}
}

// Map is a plain simple thread-safe map implementation, using a mutex for synchronization
type Map[K comparable, V any] struct {
	syncMap[K, V]
}

// Clone returns a copy of the map
func (m *Map[K, V]) Clone() Map[K, V] {
	return Map[K, V]{m.syncMap.clone()}
}

// NewMapCmp creates a new thread-safe map of type  with comparable values
// An empty map is allocated with enough space to hold the
// specified number of elements. The size may be omitted, in which case
// a small starting size is allocated.
func NewMapCmp[K, V comparable](size ...int) *MapCmp[K, V] {
	if len(size) > 0 {
		return &MapCmp[K, V]{syncMap[K, V]{items: make(map[K]V, size[0])}}
	}

	return &MapCmp[K, V]{syncMap[K, V]{items: make(map[K]V)}}
}

// MapCmp is a thread-safe map implementation with comparable values
// It extends the [Map] with additional methods for conditionally deleting or swapping values
type MapCmp[K, V comparable] struct {
	syncMap[K, V]
}

// Clone returns a copy of the map
func (m *MapCmp[K, V]) Clone() MapCmp[K, V] {
	return MapCmp[K, V]{m.syncMap.clone()}
}

// DeleteIfEqual removes the key from the map when the value is equal to the expected value
// It returns true if the key was found and removed
func (m *MapCmp[K, V]) DeleteIfEqual(key K, expected V) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.items == nil { // fast exit on empty map
		return false
	}

	m.init()
	if cur, ok := m.items[key]; ok && cur == expected {
		delete(m.items, key)
		return true
	}

	return false
}

// SwapIfEqual removes the key from the map when the value is equal to the expected value
func (m *MapCmp[K, V]) SwapIfEqual(key K, expected, val V) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.items == nil { // fast exit on empty map
		return false
	}

	m.init()
	if cur, ok := m.items[key]; ok && cur == expected {
		m.items[key] = val
		return true
	}

	return false
}

type syncMap[K comparable, V any] struct {
	mux   sync.Mutex
	items map[K]V
}

func (m *syncMap[K, V]) init() {
	if m.items != nil {
		return
	}

	m.items = make(map[K]V) // let runtime decide the needed map size
}

// Store sets a new key-value pair to the map
func (m *syncMap[K, V]) Store(key K, value V) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.init()
	m.items[key] = value
}

// Load returns the value for the key and a boolean indicating if the key was found
func (m *syncMap[K, V]) Load(key K) (V, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.init()
	value, ok := m.items[key]
	return value, ok
}

// Has checks if the map has the key
func (m *syncMap[K, V]) Has(key K) bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.init()
	_, ok := m.items[key]
	return ok
}

// Delete removes the key from the map
func (m *syncMap[K, V]) Delete(key K) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.items == nil { // fast exit on empty map
		return
	}

	m.init()
	delete(m.items, key)
}

// Len returns the number of items in the map
func (m *syncMap[K, V]) Len() int {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.init()
	return len(m.items)
}

// Keys return slice with all map keys.
func (m *syncMap[K, V]) Keys() []K {
	m.mux.Lock()
	defer m.mux.Unlock()

	var keys, i = make([]K, len(m.items)), 0
	for k := range m.items {
		keys[i], i = k, i+1
	}

	return keys
}

// Values return slice with all map values.
func (m *syncMap[K, V]) Values() []V {
	m.mux.Lock()
	defer m.mux.Unlock()

	var values, i = make([]V, len(m.items)), 0
	for _, v := range m.items {
		values[i], i = v, i+1
	}

	return values
}

// Clear removes all items from the map
func (m *syncMap[K, V]) Clear() {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.items = make(map[K]V)
}

// Swap replaces the value for the given key and returns the old value and a boolean indicating if the key was found
func (m *syncMap[K, V]) Swap(key K, val V) (V, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.init()
	cur, ok := m.items[key]
	m.items[key] = val
	return cur, ok
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *syncMap[K, V]) LoadOrStore(key K, val V) (cur V, ok bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.init()
	if cur, ok = m.items[key]; !ok {
		m.items[key], cur = val, val
	}

	return cur, ok
}

// LoadAndDelete deletes the value for a key, returning the previous value if any. The loaded result reports whether
// the key was present.
func (m *syncMap[K, V]) LoadAndDelete(key K) (cur V, ok bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.items == nil { // fast operation terminator
		return
	}

	m.init()
	if cur, ok = m.items[key]; ok {
		delete(m.items, key)
	}

	return cur, ok
}

// Iter returns an iterator for the map
//
// Iter does not necessarily correspond to any consistent snapshot of the map's contents: no key will be visited more
// than once. Iter does not block other methods on the receiver
func (m *syncMap[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.mux.Lock()

		for k, v := range m.items {
			m.mux.Unlock()
			if !yield(k, v) {
				return
			}
			m.mux.Lock()
		}

		m.mux.Unlock()
	}
}

// Snapshot returns a snapshot copy of the underlying map
func (m *syncMap[K, V]) Snapshot() map[K]V {
	m.mux.Lock()
	defer m.mux.Unlock()

	return maps.Clone(m.items)
}

// clone returns a copy of the map
func (m *syncMap[K, V]) clone() syncMap[K, V] {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.items == nil { // fast exit on empty map
		return syncMap[K, V]{}
	}

	m.init()
	return syncMap[K, V]{items: maps.Clone(m.items)}
}
