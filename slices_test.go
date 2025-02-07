package syncutils_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/esome/golang-sync"
)

func TestNewSlice(t *testing.T) {
	t.Run("size omitted", func(t *testing.T) {
		t.Parallel()

		var s = NewSlice[int]()
		require.NotNil(t, s)
		require.Equal(t, 0, s.Len())
		require.Equal(t, 0, s.Cap())
	})

	t.Run("only length given", func(t *testing.T) {
		t.Parallel()

		var s = NewSlice[int](10)
		require.NotNil(t, s)
		require.Equal(t, 10, s.Len())
		require.Equal(t, 10, s.Cap())
	})

	t.Run("length and capacity given", func(t *testing.T) {
		t.Parallel()

		var s = NewSlice[int](0, 10)
		require.NotNil(t, s)
		require.Equal(t, 0, s.Len())
		require.Equal(t, 10, s.Cap())
	})
}

func TestNewSliceFrom(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		t.Parallel()

		var s = NewSliceFrom(([]int)(nil))
		require.NotNil(t, s)
		require.Equal(t, 0, s.Len())
		require.Equal(t, 0, s.Cap())
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		var s = NewSliceFrom([]int{})
		require.NotNil(t, s)
		require.Equal(t, 0, s.Len())
		require.Equal(t, 0, s.Cap())
	})

	t.Run("non-empty slice", func(t *testing.T) {
		t.Parallel()

		items := make([]int, 3, 10)
		items[0], items[1], items[2] = 1, 2, 3

		var s = NewSliceFrom(items)
		require.NotNil(t, s)
		require.Equal(t, 3, s.Len())
		require.Equal(t, 10, s.Cap())
	})

}

func TestSlice_Grow(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		var s Slice[int]
		require.Equal(t, 0, s.Len())
		require.Equal(t, 0, s.Cap())
		require.LessOrEqual(t, 10, s.Grow(10))
		require.Equal(t, 0, s.Len())
	})

	t.Run("non-empty slice", func(t *testing.T) {
		t.Parallel()

		s := NewSliceFrom([]int{1, 2, 3})
		require.Equal(t, 3, s.Len())
		require.Equal(t, 3, s.Cap())
		require.LessOrEqual(t, 13, s.Grow(10))
		require.Equal(t, 3, s.Len())
	})
}

func TestSlice_Clip(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, 3, s.Len())
	require.Equal(t, 3, s.Cap())
	_ = s.Grow(10)
	require.Equal(t, 3, s.Clip())
	require.Equal(t, 3, s.Len())
}

func TestSlice_Clear(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, 3, s.Len())

	const ce = 3
	c := s.Clear()
	require.Conditionf(t, func() bool {
		return c >= ce && ce <= s.Cap()
	}, "capacity mismatch - expect: %d, was: %d", ce, c)
	require.Equal(t, 0, s.Len())
}

func TestSlice_Store(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, 3, s.Len())
	s.Store(1, 42)
	require.Equal(t, 42, s.At(1))
	require.Panicsf(t, func() {
		s.Store(3, 69)
	}, "Store: did not panic on index out of range")
}

func TestSlice_At(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, 3, s.Len())
	require.Equal(t, 2, s.At(1))
	require.Panicsf(t, func() {
		s.At(3)
	}, "At: did not panic on index out of range")
}

func TestSlice_Append(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, 3, s.Len())
	s.Append(4, 5, 6)
	require.Equal(t, 6, s.Len())
	require.Equal(t, 6, s.At(5))
}

func TestSlice_Concat(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, 3, s.Len())

	const le, ce = 6, 6
	l, c := s.Concat([]int{4, 5}, []int{6})
	require.Conditionf(t, func() bool {
		return l == le && le == s.Len()
	}, "length mismatch - expect: %d, was: %d", le, l)
	require.Conditionf(t, func() bool {
		return c == ce && ce == s.Cap()
	}, "capacity mismatch - expect: %d, was: %d", ce, c)
	require.Equal(t, 4, s.At(3))
	require.Equal(t, 5, s.At(4))
	require.Equal(t, 6, s.At(5))
}

func TestSlice_Delete(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3, 4, 5, 6})
	require.Equal(t, 6, s.Len())

	const le, ce = 4, 6
	l, c := s.Delete(1, 3)
	require.Conditionf(t, func() bool {
		return l == le && le == s.Len()
	}, "length mismatch - expect: %d, was: %d", le, l)
	require.Conditionf(t, func() bool {
		return c == ce && ce == s.Cap()
	}, "capacity mismatch - expect: %d, was: %d", ce, c)
	require.Panicsf(t, func() {
		_, _ = s.Delete(3, 5)
		fmt.Println(s.Snapshot())
	}, "Delete: did not panic on index out of range")
}

func TestSlice_DeleteFunc(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3, 4, 5, 6})
	require.Equal(t, 6, s.Len())

	const le, ce = 3, 6
	l, c := s.DeleteFunc(func(e int) bool {
		return e%2 == 0
	})
	require.Conditionf(t, func() bool {
		return l == le && le == s.Len()
	}, "length mismatch - expect: %d, was: %d", le, l)
	require.Conditionf(t, func() bool {
		return c == ce && ce == s.Cap()
	}, "capacity mismatch - expect: %d, was: %d", ce, c)
	require.Equal(t, 1, s.At(0))
	require.Equal(t, 3, s.At(1))
	require.Equal(t, 5, s.At(2))
}

func TestSlice_Insert(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, 3, s.Len())

	const le, ce = 5, 6
	l, c := s.Insert(1, 42, 69)
	require.Conditionf(t, func() bool {
		return l == le && le == s.Len()
	}, "length mismatch - expect: %d, was: %d", le, l)
	require.Conditionf(t, func() bool {
		return c >= ce && ce <= s.Cap()
	}, "capacity mismatch - expect: %d, was: %d", ce, c)
	require.Equal(t, 1, s.At(0))
	require.Equal(t, 42, s.At(1))
	require.Equal(t, 69, s.At(2))
	require.Equal(t, 2, s.At(3))
	require.Equal(t, 3, s.At(4))
	require.Panicsf(t, func() {
		_, _ = s.Insert(6, 1)
	}, "Insert: did not panic on index out of range")
}

func TestSlice_Replace(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3, 4, 5, 6})
	require.Equal(t, 6, s.Len())

	// case: len(v) == (j-i)
	const le, ce = 6, 6
	l, c := s.Replace(1, 3, 42, 69)
	require.Conditionf(t, func() bool {
		return l == le && le == s.Len()
	}, "length mismatch - expect: %d, was: %d", le, l)
	require.Conditionf(t, func() bool {
		return c == ce && ce == s.Cap()
	}, "capacity mismatch - expect: %d, was: %d", ce, c)
	require.Equal(t, 1, s.At(0))
	require.Equal(t, 42, s.At(1))
	require.Equal(t, 69, s.At(2))
	require.Equal(t, 4, s.At(3))
	require.Equal(t, 5, s.At(4))
	require.Equal(t, 6, s.At(5))

	// case: len(v) < (j-i)
	const le2 = 5
	l, c = s.Replace(2, 5, 127, 256)
	require.Conditionf(t, func() bool {
		return l == le2 && le2 == s.Len()
	}, "length mismatch - expect: %d, was: %d", le2, l)
	require.Conditionf(t, func() bool {
		return c == ce && ce == s.Cap()
	}, "capacity mismatch - expect: %d, was: %d", ce, c)
	require.Equal(t, 1, s.At(0))
	require.Equal(t, 42, s.At(1))
	require.Equal(t, 127, s.At(2))
	require.Equal(t, 256, s.At(3))
	require.Equal(t, 6, s.At(4))

	require.Panicsf(t, func() {
		_, _ = s.Replace(4, 6, 1)
	}, "Replace: did not panic on index out of range")
}

func TestSlice_Snapshot(t *testing.T) {
	t.Parallel()

	s := NewSliceFrom([]int{1, 2, 3})
	require.Equal(t, []int{1, 2, 3}, s.Snapshot())
}

func TestSlice_Iter(t *testing.T) {
	t.Run("concurrent change", func(t *testing.T) {
		t.Parallel()

		s := NewSliceFrom([]int{1, 2, 3})
		require.Equal(t, 3, s.Len())

		var iter int
		for range s.Iter() {
			if iter++; iter == 1 {
				s.Append(4, 5, 6)
			}
		}
		require.Equal(t, 6, iter)
	})

	t.Run("change in iter itself", func(t *testing.T) {
		t.Parallel()

		s := NewSliceFrom([]int{1, 2, 3})
		require.Equal(t, 3, s.Len())

		var iter int
		for range s.Iter() {
			if iter++; iter == 1 {
				s.Append(4, 5, 6)
			}
		}
		require.Equal(t, 6, iter)
	})
}

//go:noinline
func TestSlice_ConcurrentUsage(t *testing.T) {
	t.Parallel()

	var (
		s  = NewSlice[string](10)
		wg sync.WaitGroup
	)

	for i := 0; i < 100; i++ {
		wg.Add(13)

		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			s.Store(1, "bar")
		}() // +
		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			s.Store(0, "foo")
		}() // +
		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			_ = s.At(0)
		}()
		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			_ = s.At(1)
		}()
		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			_, _ = s.Replace(1, 1, "foobar")
		}()
		go func() {
			defer wg.Done()
			_, _ = s.DeleteFunc(func(str string) bool { return strings.Contains(str, "bar") })
		}() // -
		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			_, _ = s.Delete(1, 1)
		}() // -
		go func() {
			defer wg.Done()
			for range s.Iter() {
			}
		}()
		go func() {
			defer wg.Done()
			for range s.Iter() {
				break
			}
		}()
		go func() { defer wg.Done(); _ = s.Len() }()
		go func() { defer wg.Done(); _ = s.Cap() }()
		go func() { defer wg.Done(); _ = s.Grow(2) }()
		go func() { defer wg.Done(); _ = s.Clip() }()
	}

	wg.Wait()
}
