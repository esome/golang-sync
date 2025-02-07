package syncutils_test

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/esome/golang-sync"
)

func TestNewMap(t *testing.T) {
	t.Run("size omitted", func(t *testing.T) {
		t.Parallel()

		var m = NewMap[string, int]()
		require.NotNil(t, m)
		require.Equal(t, 0, m.Len())
	})

	t.Run("with size", func(t *testing.T) {
		t.Parallel()

		var m = NewMap[string, int](2)
		require.NotNil(t, m)
		require.Equal(t, 0, m.Len())
	})
}

func TestMap_Clone(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const key1, key2, val1, val2 = "a", "b", 111, 222

	// empty map fastpath
	var clone = m.Clone()
	require.Equal(t, 0, clone.Len())

	m.Store(key1, val1)

	require.Equal(t, 1, m.Len())
	require.EqualValues(t, []string{key1}, m.Keys())
	require.EqualValues(t, []int{val1}, m.Values())

	v, loaded := m.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val1, v)

	v, loaded = m.Load(key2)

	require.False(t, loaded)
	require.EqualValues(t, 0, v)

	clone = m.Clone()

	require.EqualValues(t, m.Len(), clone.Len())

	v, loaded = clone.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val1, v)

	m.Store(key1, val2) // overwrite in original map

	v, loaded = clone.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val1, v)

	v, loaded = m.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val2, v)
}

func TestMap_Load(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const key = "a"

	value, ok := m.Load(key) // not exists

	require.False(t, ok)
	require.EqualValues(t, 0, value)

	m.Store(key, 111)
	m.Store(key, 111)
	m.Store(key, 111) // repeated call

	value, ok = m.Load(key) // exists

	require.True(t, ok)
	require.EqualValues(t, 111, value)
}

func TestMap_Store(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const (
		key1, key2 = "a", "b"
		val1, val2 = 123, 321
	)

	m.Store(key1, val1)

	require.Equal(t, 1, m.Len())
	require.EqualValues(t, []string{key1}, m.Keys())
	require.EqualValues(t, []int{val1}, m.Values())

	m.Store(key2, val2)
	m.Store(key2, val2) // repeated call

	require.Equal(t, 2, m.Len())

	var wantKeys, gotKeys = []string{key2, key1}, m.Keys()

	sort.Strings(wantKeys)
	sort.Strings(gotKeys)
	require.EqualValues(t, wantKeys, gotKeys)

	var wantValues, gotValues = []int{val1, val2}, m.Values()

	sort.Ints(wantValues)
	sort.Ints(gotValues)
	require.EqualValues(t, wantValues, gotValues)
}

func TestMap_Swap(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const key, val1, val2 = "a", 123, 321

	v, ok := m.Swap(key, val1)

	require.False(t, ok)
	require.EqualValues(t, 0, v)
	require.Equal(t, 1, m.Len())

	v, ok = m.Swap(key, val2) // another value is passed

	require.True(t, ok)
	require.EqualValues(t, val1, v)
	require.Equal(t, 1, m.Len())
}

func TestMap_LoadOrStore(t *testing.T) {
	t.Parallel()

	var m Map[string, float64]

	const (
		key        = "a"
		val1, val2 = 123.123, 321.321
	)

	v, loaded := m.LoadOrStore(key, val1)

	require.False(t, loaded)
	require.EqualValues(t, val1, v)
	require.Equal(t, 1, m.Len())

	v, loaded = m.LoadOrStore(key, val2) // another value is passed

	require.True(t, loaded)
	require.EqualValues(t, val1, v)
	require.Equal(t, 1, m.Len())
}

//go:noinline
func TestMap_LoadAndDelete(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const key, val = "a", 123

	v, loaded := m.LoadAndDelete(key)
	require.False(t, loaded)
	require.EqualValues(t, 0, v)

	m.Store(key, val)

	require.Equal(t, 1, m.Len())
	require.EqualValues(t, []string{key}, m.Keys())
	require.EqualValues(t, []int{val}, m.Values())

	v, loaded = m.LoadAndDelete(key)
	require.True(t, loaded)
	require.EqualValues(t, val, v)

	require.Equal(t, 0, m.Len())
	require.EqualValues(t, []string{}, m.Keys())
	require.EqualValues(t, []int{}, m.Values())

	_, _ = m.LoadAndDelete(key)
	_, _ = m.LoadAndDelete(key)
	v, loaded = m.LoadAndDelete(key) // repeated call

	require.False(t, loaded)
	require.EqualValues(t, 0, v)

	require.Equal(t, 0, m.Len())
	require.EqualValues(t, []string{}, m.Keys())
	require.EqualValues(t, []int{}, m.Values())
}

func TestMap_Delete(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const key, val = "a", 123

	m.Delete(key)
	m.Delete(key) // repeated call

	require.Equal(t, 0, m.Len())
	require.EqualValues(t, []string{}, m.Keys())
	require.EqualValues(t, []int{}, m.Values())

	m.Store(key, val)

	require.Equal(t, 1, m.Len())
	require.EqualValues(t, []string{key}, m.Keys())
	require.EqualValues(t, []int{val}, m.Values())

	m.Delete(key)
	m.Delete(key)
	m.Delete(key) // repeated call

	require.Equal(t, 0, m.Len())
	require.EqualValues(t, []string{}, m.Keys())
	require.EqualValues(t, []int{}, m.Values())
}

func TestMap_Iter(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const (
		key1, key2             = "a", "b"
		val1, val2, val3, val4 = 123, 321, 456, 654
	)

	var iter uint
	for range m.Iter() {
		iter++
		break
	}

	require.EqualValues(t, 0, iter)

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			m.Store(key1, val1)
			m.Store(key2, val2)
		}()
	}
	wg.Wait()

	require.Equal(t, 2, m.Len())

	iter = 0 // reset
	for key, val := range m.Iter() {
		switch key {
		case key1:
			m.Swap(key1, val3)
			require.EqualValues(t, val1, val)
			k1v, _ := m.Load(key1)
			require.EqualValues(t, val3, k1v)
		case key2:
			m.Swap(key2, val4)
			require.EqualValues(t, val2, val)
			k2v, _ := m.Load(key2)
			require.EqualValues(t, val4, k2v)
		}
		iter++
	}

	require.EqualValues(t, 2, iter)

	k1v, _ := m.Load(key1)
	require.EqualValues(t, val3, k1v)
	k2v, _ := m.Load(key2)
	require.EqualValues(t, val4, k2v)

	iter = 0 // reset
	for range m.Iter() {
		iter++
		break
	}

	require.EqualValues(t, 1, iter)
}

func TestMap_Snapshot(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const key1, key2 = "a", "b"
	const val1, val2 = 123, 321

	m.Store(key1, val1)
	m.Store(key2, val2)

	require.EqualValues(t, map[string]int{key1: val1, key2: val2}, m.Snapshot())
}

func TestMap_Clear(t *testing.T) {
	t.Parallel()

	var m Map[string, int]

	const key1, key2 = "a", "b"
	const val1, val2 = 123, 321

	m.Store(key1, val1)
	m.Store(key2, val2)

	require.Equal(t, 2, m.Len())
	require.True(t, m.Has(key1))
	require.True(t, m.Has(key2))

	m.Clear()
	require.Equal(t, 0, m.Len())
	require.False(t, m.Has(key1))
	require.False(t, m.Has(key2))
}

func TestNewMapCmp(t *testing.T) {
	t.Run("size omitted", func(t *testing.T) {
		t.Parallel()

		var m = NewMapCmp[string, int]()
		require.NotNil(t, m)
		require.Equal(t, 0, m.Len())
	})

	t.Run("with size", func(t *testing.T) {
		t.Parallel()

		var m = NewMapCmp[string, int](2)
		require.NotNil(t, m)
		require.Equal(t, 0, m.Len())
	})
}

func TestMapCmp_Clone(t *testing.T) {
	t.Parallel()

	var m MapCmp[string, int]

	const key1, key2, val1, val2 = "a", "b", 111, 222

	// empty map fastpath
	var clone = m.Clone()
	require.Equal(t, 0, clone.Len())

	m.Store(key1, val1)

	require.Equal(t, 1, m.Len())
	require.EqualValues(t, []string{key1}, m.Keys())
	require.EqualValues(t, []int{val1}, m.Values())

	v, loaded := m.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val1, v)

	v, loaded = m.Load(key2)

	require.False(t, loaded)
	require.EqualValues(t, 0, v)

	clone = m.Clone()

	require.EqualValues(t, m.Len(), clone.Len())

	v, loaded = clone.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val1, v)

	m.Store(key1, val2) // overwrite in original map

	v, loaded = clone.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val1, v)

	v, loaded = m.Load(key1)

	require.True(t, loaded)
	require.EqualValues(t, val2, v)
}

func TestMapCmp_DeleteIfEqual(t *testing.T) {
	t.Parallel()

	var m MapCmp[string, int]

	const key1, key2 = "a", "b"
	const val1, val2, val3 = 123, 321, 456

	// empty map fastpath
	require.False(t, m.DeleteIfEqual(key1, val1))

	m.Store(key1, val1)
	m.Store(key2, val2)

	require.Equal(t, 2, m.Len())
	require.True(t, m.Has(key1))
	require.True(t, m.Has(key2))

	require.False(t, m.DeleteIfEqual(key1, val3))
	require.Equal(t, 2, m.Len())
	require.True(t, m.Has(key1))

	require.True(t, m.DeleteIfEqual(key2, val2))
	require.Equal(t, 1, m.Len())
	require.False(t, m.Has(key2))
}

func TestMapCmp_SwapIfEqual(t *testing.T) {
	t.Parallel()

	var m MapCmp[string, int]

	const key1, key2 = "a", "b"
	const val1, val2, val3, val4 = 123, 321, 456, 654

	// empty map fastpath
	require.False(t, m.SwapIfEqual(key1, val3, val4))

	m.Store(key1, val1)
	m.Store(key2, val2)

	require.Equal(t, 2, m.Len())
	require.True(t, m.Has(key1))
	require.True(t, m.Has(key2))

	require.False(t, m.SwapIfEqual(key1, val3, val4))
	cur, _ := m.Load(key1)
	require.Equal(t, val1, cur)

	require.True(t, m.SwapIfEqual(key2, val2, val4))
	cur, _ = m.Load(key2)
	require.Equal(t, val4, cur)
}

//go:noinline
func TestMap_ConcurrentUsage(t *testing.T) { // race detector provocation
	t.Parallel()

	var (
		m  Map[string, int]
		wg sync.WaitGroup
	)

	for i := 0; i < 100; i++ {
		wg.Add(14)

		go func() { defer wg.Done(); _, _ = m.LoadOrStore("foo", 1) }() // +
		go func() { defer wg.Done(); m.Store("foo", 1) }()              // +
		go func() { defer wg.Done(); _, _ = m.Load("foo") }()
		go func() { defer wg.Done(); _ = m.Has("foo") }()
		go func() { defer wg.Done(); _ = m.Has("bar") }()
		go func() { defer wg.Done(); _, _ = m.Swap("foo", 2) }()
		go func() { defer wg.Done(); _, _ = m.LoadAndDelete("foo") }() // -
		go func() { defer wg.Done(); m.Delete("foo") }()               // -
		go func() {
			defer wg.Done()
			for range m.Iter() {
			}
		}()
		go func() {
			defer wg.Done()
			for range m.Iter() {
				break
			}
		}()
		go func() { defer wg.Done(); _ = m.Len() }()
		go func() { defer wg.Done(); _ = m.Keys() }()
		go func() { defer wg.Done(); _ = m.Values() }()
		go func() { defer wg.Done(); _ = m.Clone() }()
	}

	wg.Wait()
}
