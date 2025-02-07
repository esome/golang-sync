// Package syncutils provides thread-safe data structures and other utilities for working with concurrency.

package syncutils

import (
	"iter"
	"slices"
	"sync"
)

// NewSlice creates a new thread-safe slice of type E
// The size specifies the length. The capacity of the slice is
// equal to its length. A second integer argument may be provided to
// specify a different capacity; it must be no smaller than the
// length. For example, make([] int, 0, 10) allocates an underlying array
// of size 10 and returns a slice of length 0 and capacity 10 that is
// backed by this underlying array.
func NewSlice[E any](sizes ...int) *Slice[E] {
	switch len(sizes) {
	case 0:
		return &Slice[E]{}
	case 1:
		return &Slice[E]{elements: make([]E, sizes[0])}
	default:
		return &Slice[E]{elements: make([]E, sizes[0], sizes[1])}
	}
}

// NewSliceFrom creates a new thread-safe slice of type E from the given elements
// The size of the slice is equal to the length of the elements, as well as the capacity
// The elements are copied into the resulting Slice
func NewSliceFrom[E any](elements []E) *Slice[E] {
	s := NewSlice[E](len(elements), cap(elements))
	copy(s.elements, elements)
	return s
}

// Slice is a thread-safe slice implementation
type Slice[E any] struct {
	mux      sync.Mutex
	elements []E
}

// Len returns the length of the slice
func (s *Slice[E]) Len() int {
	s.mux.Lock()
	defer s.mux.Unlock()

	return len(s.elements)
}

// Cap returns the capacity of the slice
func (s *Slice[E]) Cap() int {
	s.mux.Lock()
	defer s.mux.Unlock()

	return cap(s.elements)
}

// Grow increases the slice's capacity, if necessary, to guarantee space for another n elements.
// After Grow(n), at least n elements can be appended to the slice without another allocation.
// If n is negative or too large to allocate the memory, Grow panics.
// It returns the new capacity of the slice.
func (s *Slice[E]) Grow(n int) int {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = slices.Grow(s.elements, n)
	return cap(s.elements)
}

// Clear removes all elements from the slice but maintains the current capacity, returning the capacity of the slice.
func (s *Slice[E]) Clear() int {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = s.elements[:0:cap(s.elements)]
	return cap(s.elements)
}

// Clip removes unused capacity from the slice, returning the new capacity.
func (s *Slice[E]) Clip() int {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = slices.Clip(s.elements)
	return cap(s.elements)
}

// Store sets the element at the given index to the given value.
// If the index is out of range, Store panics.
func (s *Slice[E]) Store(i int, val E) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements[i] = val
}

// At returns the element at the given index.
// If the index is out of range, At panics.
func (s *Slice[E]) At(i int) E {
	s.mux.Lock()
	defer s.mux.Unlock()

	return s.elements[i]
}

// Append appends new items to the slice.
// The new items are added to the end of the slice.
func (s *Slice[E]) Append(items ...E) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = append(s.elements, items...)
}

// Concat concatenates the passed in slices to the current one,
// returning the new length and capacity of the slice.
func (s *Slice[E]) Concat(other ...[]E) (int, int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	items := make([][]E, 1, len(other)+1)
	items[0] = s.elements
	items = append(items, other...)
	s.elements = slices.Concat(items...)
	return len(s.elements), cap(s.elements)
}

// Delete removes the elements s[i:j] from s, returning the new length and capacity of the slice.
// Delete panics if j > len(s) or s[i:j] is not a valid slice of s.
// Delete is O(len(s)-i), so if many items must be deleted, it is better to
// make a single call deleting them all together than to delete one at a time.
// Delete zeroes the elements s[len(s)-(j-i):len(s)].
func (s *Slice[E]) Delete(i, j int) (int, int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = slices.Delete(s.elements, i, j)
	return len(s.elements), cap(s.elements)
}

// DeleteFunc removes any elements from s for which del returns true,
// returning the new length and capacity of the slice.
// DeleteFunc zeroes the elements between the new length and the original length.
func (s *Slice[E]) DeleteFunc(del func(E) bool) (int, int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = slices.DeleteFunc(s.elements, del)
	return len(s.elements), cap(s.elements)
}

// Insert inserts the values v... into s at index i, and returns the new length and capacity of the slice.
// The elements at s[i:] are shifted up to make room.
// In the resulting slice r, r[i] == v[0],
// and r[i+len(v)] == value originally at r[i].
// Insert panics if i is out of range.
// This function is O(len(s) + len(v)).
func (s *Slice[E]) Insert(i int, v ...E) (int, int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = slices.Insert(s.elements, i, v...)
	return len(s.elements), cap(s.elements)
}

// Replace replaces the elements s[i:j] by the given v, returning the new length and capacity of the slice.
// Replace panics if j > len(s) or s[i:j] is not a valid slice of s.
// When len(v) < (j-i), Replace zeroes the elements between the new length and the original length.
func (s *Slice[E]) Replace(i, j int, v ...E) (int, int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.elements = slices.Replace(s.elements, i, j, v...)
	return len(s.elements), cap(s.elements)
}

// Snapshot returns a copy of the slice
func (s *Slice[E]) Snapshot() []E {
	s.mux.Lock()
	defer s.mux.Unlock()

	out := make([]E, len(s.elements))
	copy(out, s.elements)
	return out
}

// Iter returns an iterator for the slice
//
// Iter does not necessarily correspond to any consistent snapshot of the slice's contents: no key will be visited more
// than once. Iter does not block other methods on the receiver
func (s *Slice[E]) Iter() iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		s.mux.Lock()

		var i int
		for {
			if i >= len(s.elements) {
				break
			}

			// we need to reference the element before unlocking the mutex to avoid data races
			val := s.elements[i]
			s.mux.Unlock()
			if !yield(i, val) {
				return
			}
			i++
			s.mux.Lock()
		}

		s.mux.Unlock()
	}
}
