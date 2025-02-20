package channels_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	. "github.com/esome/golang-sync/channels"
)

func TestChunkAndDo(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("immediately exits on size 0", func(t *testing.T) {
		src := make(chan struct{})
		err := ChunkAndDo(context.TODO(), src, 0, func(context.Context, []struct{}) error { return nil })
		assert.EqualError(t, err, "parameter size must be greater than 0")
	})

	t.Run("success", func(t *testing.T) {
		sizes := []int{10, 15}
		for _, size := range sizes {
			t.Run(fmt.Sprintf("chunk-size: %d", size), func(t *testing.T) {
				src := make(chan int, 100)
				go func() {
					for i := 0; i < 100; i++ {
						src <- i
					}
					close(src)
				}()

				var total int
				err := ChunkAndDo(context.TODO(), src, uint(size), func(ctx context.Context, c []int) error {
					for _, j := range c {
						total += j
					}
					if !assert.LessOrEqualf(t, len(c), size,
						"method was not expected to be called with more than %d elements", size,
					) {
						return fmt.Errorf("call with unexpected chunk size")
					}
					return ctx.Err()
				})
				assert.NoError(t, err)
				assert.Equal(t, 4950, total)
			})
		}
	})

	t.Run("error from doer", func(t *testing.T) {
		src := make(chan int, 100)
		go func() {
			for i := 0; i < 100; i++ {
				src <- i
			}
			close(src)
		}()

		const size = 10
		var total, calls int
		err := ChunkAndDo(context.TODO(), src, size, func(ctx context.Context, c []int) error {
			for _, j := range c {
				total += j
			}
			if calls++; calls >= 6 {
				return fmt.Errorf("method must not be called more often than 5 times")
			}

			return ctx.Err()
		})
		assert.EqualError(t, err, "method must not be called more often than 5 times")
		assert.Equal(t, 1770, total)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		src := make(chan int)
		go func() {
			for i := 0; i < 100; i++ {
				if err := Send(ctx, src, i); err != nil {
					return
				}
				time.Sleep(5 * time.Microsecond)
			}
			close(src)
		}()

		const size = 10
		var total, calls int
		err := ChunkAndDo(ctx, src, size, func(ctx context.Context, c []int) error {
			for _, j := range c {
				total += j
			}
			if calls++; calls >= 6 {
				cancel()
				return nil
			}

			return ctx.Err()
		})
		assert.EqualError(t, err, context.Canceled.Error())
		assert.Equal(t, 1770, total)
	})
}

func TestCollect(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("success", func(t *testing.T) {
		refs := make([]int, 100)
		for i := 0; i < 100; i++ {
			refs[i] = i
		}

		src := make(chan int, len(refs))
		go func() {
			for _, i := range refs {
				src <- i
			}
			close(src)
		}()

		ints, err := Collect(context.TODO(), src)
		assert.NoError(t, err)
		assert.Equal(t, refs, ints)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Millisecond)
		defer cancel()

		src := make(chan int)
		go func() {
			defer close(src)
			tick := time.Tick(100 * time.Microsecond)
			for i := 0; i < 100; i++ {
				select {
				case <-ctx.Done():
					return
				case <-tick:
					Send(ctx, src, i)
				}
			}
		}()

		ints, err := Collect(ctx, src)
		assert.ErrorIsf(t, err, context.DeadlineExceeded,
			"context was expected to be canceled, but is: %q", err)
		assert.NotEmpty(t, ints)
		assert.Greater(t, 50, len(ints))
	})
}

func TestDrain(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("success", func(t *testing.T) {
		refs := make([]int, 100)
		for i := 0; i < 100; i++ {
			refs[i] = i
		}

		src := make(chan int, len(refs))
		go func() {
			for _, i := range refs {
				src <- i
			}
			close(src)
		}()

		err := Drain(context.TODO(), src)
		assert.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond)
		defer cancel()

		src := make(chan int)
		go func() {
			defer close(src)
			tick := time.Tick(15 * time.Microsecond)
			for i := 0; i < 100; i++ {
				select {
				case <-ctx.Done():
					return
				case <-tick:
					Send(ctx, src, i)
				}
			}
		}()

		err := Drain(ctx, src)
		assert.ErrorIsf(t, err, context.DeadlineExceeded,
			"context was expected to be canceled, but is: %q", err)
	})
}

func TestDrainWithCount(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("success", func(t *testing.T) {
		refs := make([]int, 100)
		for i := 0; i < 100; i++ {
			refs[i] = i
		}

		src := make(chan int, len(refs))
		go func() {
			for _, i := range refs {
				src <- i
			}
			close(src)
		}()

		discarded, err := DrainWithCount(context.TODO(), src)
		assert.NoError(t, err)
		assert.Equal(t, 100, discarded)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Millisecond)
		defer cancel()

		src := make(chan int)
		go func() {
			defer close(src)
			tick := time.Tick(100 * time.Microsecond) // 5000 / 100 = 50 at max
			for i := 0; i < 100; i++ {
				select {
				case <-ctx.Done():
					return
				case <-tick:
					Send(ctx, src, i)
				}
			}
		}()

		discarded, err := DrainWithCount(ctx, src)
		assert.ErrorIsf(t, err, context.DeadlineExceeded,
			"context was expected to be canceled, but is: %q", err)
		assert.NotEmpty(t, discarded)
		assert.Greater(t, 50, discarded)
	})
}

func TestMerge(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("success", func(t *testing.T) {
		srcs := make([]<-chan int, 10)
		for i := range srcs {
			if i%2 == 1 { // skip every second source
				srcs[i] = nil
				continue
			}

			srcs[i] = func(i int) <-chan int {
				ctx := context.TODO()
				src := make(chan int, 10)
				go func() {
					defer close(src)
					for j := 0; j < 10; j++ {
						_ = Send(ctx, src, i*10+j)
					}
				}()
				return src
			}(i)
		}

		ctx := context.TODO()
		merged := Merge(ctx, srcs...)
		ints, err := Collect(ctx, merged)
		assert.NoError(t, err)
		assert.Len(t, ints, 50)
	})

	t.Run("context cancellation", func(t *testing.T) {
		sendCtx, cancel := context.WithCancel(context.TODO())
		t.Cleanup(cancel)

		srcs := make([]<-chan int, 10)
		for i := range srcs {
			srcs[i] = func(i int) <-chan int {
				src := make(chan int, 10)
				go func() {
					defer close(src)
					for j := 0; j < 10; j++ {
						_ = Send(sendCtx, src, i*10+j)
					}
				}()
				return src
			}(i)
		}

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		merged := Merge(ctx, srcs...)
		var items = make([]int, 0, 50)
		for nmr := range merged {
			items = append(items, nmr)
			if len(items) == 25 {
				cancel()
				time.Sleep(5 * time.Millisecond) // give the merge goroutine some time to exit
			}
		}

		assert.Greater(t, cap(items), len(items))
	})
}
