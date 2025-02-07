package channels_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	. "github.com/esome/golang-sync/channels"
)

func TestBroadcast(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("immediately returns when called with no receivers", func(t *testing.T) {
		err := Broadcast(context.TODO(), make(chan struct{}))
		assert.ErrorIs(t, err, ErrBroadcastWithNoReceivers)
	})

	t.Run("success", func(t *testing.T) {
		src := make(chan int, 10)
		ref := make([]int, 10)
		for i := 0; i < 10; i++ {
			src <- i
			ref[i] = i
		}
		close(src)

		// two receivers with different buffer sizes
		dsts := [2]chan int{
			make(chan int, 5),
			make(chan int),
		}

		go func() {
			assert.NoError(t, Broadcast(context.TODO(), src, dsts[0], dsts[1]))
			close(dsts[0])
			close(dsts[1])
		}()

		ints := [2][]int{nil, nil}
		for dsts[0] != nil || dsts[1] != nil {
			select {
			case nmr, ok := <-dsts[0]:
				if !ok {
					dsts[0] = nil
					continue
				}
				ints[0] = append(ints[0], nmr)
			case nmr, ok := <-dsts[1]:
				if !ok {
					dsts[1] = nil
					continue
				}
				ints[1] = append(ints[1], nmr)
			}
		}

		assert.Equalf(t, ref, ints[0], "elements received on channel: 1 are unexpected")
		assert.Equalf(t, ref, ints[1], "elements received on channel: 2 are unexpected")
	})

	t.Run("context cancellation", func(t *testing.T) {
		ref := make([]int, 10)
		for i := 0; i < 10; i++ {
			ref[i] = i
		}

		src := make(chan int, 10)
		go func() {
			for i := 0; i < 10; i++ {
				src <- i
				time.Sleep(100 * time.Microsecond)
			}
			close(src)
		}()

		// two receivers with different buffer sizes
		dsts := [2]chan int{
			make(chan int, 5),
			make(chan int),
		}

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		go func(d1, d2 chan<- int) {
			assert.ErrorIs(t, Broadcast(ctx, src, d1, d2), context.Canceled)
			close(d1)
			close(d2)
		}(dsts[0], dsts[1])

		ints := [2][]int{nil, nil}
		for dsts[0] != nil || dsts[1] != nil {
			select {
			case nmr, ok := <-dsts[0]:
				if !ok {
					dsts[0] = nil
					continue
				}
				ints[0] = append(ints[0], nmr)
				if len(ints[0]) == 5 {
					cancel()
					dsts[0] = nil
				}
			case nmr, ok := <-dsts[1]:
				if !ok {
					dsts[1] = nil
					continue
				}
				ints[1] = append(ints[1], nmr)
			}
		}

		assert.Equalf(t, ref[:5], ints[0], "elements received on channel #1 are unexpected")
		assert.Condition(t, func() (success bool) {
			size := len(ints[1])
			return assert.GreaterOrEqual(t, 6, size, "channel #2 is not expected to receive more than 6 elements") &&
				assert.Equal(t, ref[:size], ints[1], "elements received on channel #2 are unexpected")
		})
	})

	t.Run("called with nil source", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Millisecond)
		defer cancel()

		var src chan int
		dsts := [2]chan int{
			make(chan int, 5),
			make(chan int),
		}

		go func() {
			assert.NoError(t, Broadcast(ctx, src, dsts[0], dsts[1]))
			close(dsts[0])
			close(dsts[1])
		}()

		ints := [2][]int{nil, nil}
		for dsts[0] != nil || dsts[1] != nil {
			select {
			case nmr, ok := <-dsts[0]:
				if !ok {
					dsts[0] = nil
					continue
				}
				ints[0] = append(ints[0], nmr)
			case nmr, ok := <-dsts[1]:
				if !ok {
					dsts[1] = nil
					continue
				}
				ints[1] = append(ints[1], nmr)
			}
		}

		assert.Emptyf(t, ints[0], "elements received on channel: 1 are unexpected")
		assert.Emptyf(t, ints[1], "elements received on channel: 2 are unexpected")
	})
}

func TestBroadcastWithCount(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("immediately returns when called with no receivers", func(t *testing.T) {
		_, err := BroadcastWithCount(context.TODO(), make(chan struct{}))
		assert.ErrorIs(t, err, ErrBroadcastWithNoReceivers)
	})

	t.Run("success", func(t *testing.T) {
		src := make(chan int, 10)
		ref := make([]int, 10)
		for i := 0; i < 10; i++ {
			src <- i
			ref[i] = i
		}
		close(src)

		// two receivers with different buffer sizes
		dsts := [2]chan int{
			make(chan int, 5),
			make(chan int),
		}

		var sent int
		go func() {
			var err error
			sent, err = BroadcastWithCount(context.TODO(), src, dsts[0], dsts[1])
			assert.NoError(t, err)
			close(dsts[0])
			close(dsts[1])
		}()

		ints := [2][]int{nil, nil}
		for dsts[0] != nil || dsts[1] != nil {
			select {
			case nmr, ok := <-dsts[0]:
				if !ok {
					dsts[0] = nil
					continue
				}
				ints[0] = append(ints[0], nmr)
			case nmr, ok := <-dsts[1]:
				if !ok {
					dsts[1] = nil
					continue
				}
				ints[1] = append(ints[1], nmr)
			}
		}

		assert.Equalf(t, len(ref), sent, "number of sent elements is unexpected")
		assert.Equalf(t, ref, ints[0], "elements received on channel: 1 are unexpected")
		assert.Equalf(t, ref, ints[1], "elements received on channel: 2 are unexpected")
	})

	t.Run("context cancellation", func(t *testing.T) {
		ref := make([]int, 10)
		for i := 0; i < 10; i++ {
			ref[i] = i
		}

		src := make(chan int, 10)
		go func() {
			for i := 0; i < 10; i++ {
				src <- i
				time.Sleep(100 * time.Microsecond)
			}
			close(src)
		}()

		// two receivers with different buffer sizes
		dsts := [2]chan int{
			make(chan int, 5),
			make(chan int),
		}

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		var sent int
		go func(d1, d2 chan<- int) {
			var err error
			sent, err = BroadcastWithCount(ctx, src, dsts[0], dsts[1])
			assert.ErrorIs(t, err, context.Canceled)
			close(d1)
			close(d2)
		}(dsts[0], dsts[1])

		ints := [2][]int{nil, nil}
		for dsts[0] != nil || dsts[1] != nil {
			select {
			case nmr, ok := <-dsts[0]:
				if !ok {
					dsts[0] = nil
					continue
				}
				ints[0] = append(ints[0], nmr)
				if len(ints[0]) == 5 {
					cancel()
					dsts[0] = nil
				}
			case nmr, ok := <-dsts[1]:
				if !ok {
					dsts[1] = nil
					continue
				}
				ints[1] = append(ints[1], nmr)
			}
		}

		assert.Equalf(t, len(ints[0]), sent, "number of sent elements is unexpected")
		assert.Equalf(t, ref[:5], ints[0], "elements received on channel #1 are unexpected")
		assert.Condition(t, func() (success bool) {
			size := len(ints[1])
			return assert.LessOrEqual(t, sent, size, "channel #2 is not expected to receive less than 5 elements") &&
				assert.Greater(t, len(ref), size, "channel #2 is not expected to receive more than 6 elements") &&
				assert.Equal(t, ref[:size], ints[1], "elements received on channel #2 are unexpected")
		})
	})

	t.Run("called with nil source", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Millisecond)
		defer cancel()

		var src chan int
		dsts := [2]chan int{
			make(chan int, 5),
			make(chan int),
		}

		var sent int
		go func() {
			var err error
			sent, err = BroadcastWithCount(ctx, src, dsts[0], dsts[1])
			assert.NoError(t, err)
			close(dsts[0])
			close(dsts[1])
		}()

		ints := [2][]int{nil, nil}
		for dsts[0] != nil || dsts[1] != nil {
			select {
			case nmr, ok := <-dsts[0]:
				if !ok {
					dsts[0] = nil
					continue
				}
				ints[0] = append(ints[0], nmr)
			case nmr, ok := <-dsts[1]:
				if !ok {
					dsts[1] = nil
					continue
				}
				ints[1] = append(ints[1], nmr)
			}
		}

		assert.Equalf(t, 0, sent, "number of sent elements is unexpected")
		assert.Emptyf(t, ints[0], "elements received on channel: 1 are unexpected")
		assert.Emptyf(t, ints[1], "elements received on channel: 2 are unexpected")
	})
}

func TestForward(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("no errors", func(t *testing.T) {
		numbers, errs := make(chan int), make(chan error)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
			close(errs)
		}()

		go func() {
			Forward(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 0; i < 50; i++ {
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			wg.Done()
		}()

		go func() {
			Forward(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 50; i < 100; i++ {
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			wg.Done()
		}()

		var ints []int
		for numbers != nil || errs != nil {
			select {
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				t.Errorf("Forward(): must not yield an error here: %s", err)
			case j, ok := <-numbers:
				if !ok {
					numbers = nil
					continue
				}
				ints = append(ints, j)
			}
		}

		ref := make([]int, 100)
		for x := 0; x < 100; x++ {
			ref[x] = x
		}

		assert.ElementsMatch(t, ref, ints)
	})

	t.Run("with errors", func(t *testing.T) {
		numbers, errs := make(chan int), make(chan error)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
			close(errs)
		}()

		go func() {
			Forward(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 0; i < 50; i++ {
						if i%5 == 0 {
							errC <- fmt.Errorf("number must not be dividable by 5")
							continue
						}
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			wg.Done()
		}()

		go func() {
			Forward(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 50; i < 100; i++ {
						if i%7 == 0 {
							errC <- fmt.Errorf("number must not be dividable by 7")
							continue
						}
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			wg.Done()
		}()

		var ints []int
		var errList []error
		for numbers != nil || errs != nil {
			select {
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				errList = append(errList, err)
			case j, ok := <-numbers:
				if !ok {
					numbers = nil
					continue
				}
				ints = append(ints, j)
			}
		}

		ref := make([]int, 0, 83)
		for x := 0; x < 50; x++ {
			if x%5 == 0 {
				continue
			}
			ref = append(ref, x)
		}
		for x := 50; x < 100; x++ {
			if x%7 == 0 {
				continue
			}
			ref = append(ref, x)
		}

		assert.ElementsMatch(t, ref, ints)
		assert.Len(t, errList, 17)
	})

	t.Run("context cancellation", func(t *testing.T) {
		numbers, errs := make(chan int), make(chan error)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
			close(errs)
		}()

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		go func() {
			err := Forward(ctx, func(ctx1 context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 0; i < 50; i++ {
						if i%5 == 0 {
							SendE(ctx1, errC, fmt.Errorf("number must not be dividable by 5"))
							continue
						}
						Send(ctx1, intC, i)
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.EqualError(t, err, context.Canceled.Error())
			wg.Done()
		}()

		go func() {
			err := Forward(ctx, func(ctx2 context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 50; i < 100; i++ {
						if i%7 == 0 {
							SendE(ctx2, errC, fmt.Errorf("number must not be dividable by 7"))
							continue
						}
						Send(ctx2, intC, i)
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.EqualError(t, err, context.Canceled.Error())
			wg.Done()
		}()

		var errList []error
		for numbers != nil || errs != nil {
			select {
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				errList = append(errList, err)
				if len(errList) == 3 {
					cancel()
					time.Sleep(50 * time.Microsecond)
				}
			case _, ok := <-numbers:
				if !ok {
					numbers = nil
					continue
				}
			}
		}

		assert.Len(t, errList, 3)
	})

	t.Run("without error channel", func(t *testing.T) {
		numbers := make(chan int)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
		}()

		go func() {
			Forward(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC := make(chan int)
				go func() {
					for i := 0; i < 50; i++ {
						intC <- i
					}
					close(intC)
				}()
				return intC, nil
			}, numbers, nil)
			wg.Done()
		}()

		go func() {
			Forward(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC := make(chan int)
				go func() {
					for i := 50; i < 100; i++ {
						intC <- i
					}
					close(intC)
				}()
				return intC, nil
			}, numbers, nil)
			wg.Done()
		}()

		var ints []int
		for numbers != nil {
			j, ok := <-numbers
			if !ok {
				numbers = nil
				continue
			}
			ints = append(ints, j)
		}

		ref := make([]int, 100)
		for x := 0; x < 100; x++ {
			ref[x] = x
		}

		assert.ElementsMatch(t, ref, ints)
	})
}

func TestForwardWithCounts(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("no errors", func(t *testing.T) {
		numbers, errs := make(chan int), make(chan error)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
			close(errs)
		}()

		var counts [2][2]int
		go func(counts *[2]int) {
			var err error
			counts[0], counts[1], err = ForwardWithCounts(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 0; i < 50; i++ {
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.NoError(t, err)
			wg.Done()
		}(&counts[0])

		go func(counts *[2]int) {
			var err error
			counts[0], counts[1], err = ForwardWithCounts(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 50; i < 100; i++ {
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.NoError(t, err)
			wg.Done()
		}(&counts[1])

		var ints []int
		for numbers != nil || errs != nil {
			select {
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				t.Errorf("ForwardWithCounts(): must not yield an error here: %s", err)
			case j, ok := <-numbers:
				if !ok {
					numbers = nil
					continue
				}
				ints = append(ints, j)
			}
		}

		ref := make([]int, 100)
		for x := 0; x < 100; x++ {
			ref[x] = x
		}

		assert.ElementsMatch(t, ref, ints)
		assert.Equal(t, 50, counts[0][0])
		assert.Zero(t, counts[0][1])
		assert.Equal(t, 50, counts[1][0])
		assert.Zero(t, counts[1][1])
	})

	t.Run("with errors", func(t *testing.T) {
		numbers, errs := make(chan int), make(chan error)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
			close(errs)
		}()

		var counts [2][2]int
		go func(counts *[2]int) {
			var err error
			counts[0], counts[1], err = ForwardWithCounts(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 0; i < 50; i++ {
						if i%5 == 0 {
							errC <- fmt.Errorf("number must not be dividable by 5")
							continue
						}
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.NoError(t, err)
			wg.Done()
		}(&counts[0])

		go func(counts *[2]int) {
			var err error
			counts[0], counts[1], err = ForwardWithCounts(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					for i := 50; i < 100; i++ {
						if i%7 == 0 {
							errC <- fmt.Errorf("number must not be dividable by 7")
							continue
						}
						intC <- i
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.NoError(t, err)
			wg.Done()
		}(&counts[1])

		var ints []int
		var errList []error
		for numbers != nil || errs != nil {
			select {
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				errList = append(errList, err)
			case j, ok := <-numbers:
				if !ok {
					numbers = nil
					continue
				}
				ints = append(ints, j)
			}
		}

		ref := make([]int, 0, 83)
		for x := 0; x < 50; x++ {
			if x%5 == 0 {
				continue
			}
			ref = append(ref, x)
		}
		for x := 50; x < 100; x++ {
			if x%7 == 0 {
				continue
			}
			ref = append(ref, x)
		}

		assert.ElementsMatch(t, ref, ints)
		assert.Len(t, errList, 17)
		assert.Equal(t, 40, counts[0][0])
		assert.Equal(t, 10, counts[0][1])
		assert.Equal(t, 43, counts[1][0])
		assert.Equal(t, 7, counts[1][1])
	})

	t.Run("context cancellation", func(t *testing.T) {
		numbers, errs := make(chan int), make(chan error)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
			close(errs)
		}()

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		var counts [2][2]int
		go func(counts *[2]int) {
			var err error
			counts[0], counts[1], err = ForwardWithCounts(ctx, func(ctx1 context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					var errsPushed int
					for i := 0; i < 50; i++ {
						if i%5 == 0 && errsPushed < 2 {
							_ = SendE(ctx1, errC, fmt.Errorf("number must not be dividable by 5"))
							errsPushed++
							continue
						}
						_ = Send(ctx1, intC, i)
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.ErrorIs(t, err, context.Canceled)
			wg.Done()
		}(&counts[0])

		go func(counts *[2]int) {
			var err error
			counts[0], counts[1], err = ForwardWithCounts(ctx, func(ctx2 context.Context) (<-chan int, <-chan error) {
				intC, errC := make(chan int), make(chan error)
				go func() {
					var errsPushed int
					for i := 50; i < 100; i++ {
						if i%7 == 0 && errsPushed < 2 {
							_ = SendE(ctx2, errC, fmt.Errorf("number must not be dividable by 7"))
							errsPushed++
							continue
						}
						_ = Send(ctx2, intC, i)
					}
					close(intC)
					close(errC)
				}()
				return intC, errC
			}, numbers, errs)
			assert.ErrorIs(t, err, context.Canceled)
			wg.Done()
		}(&counts[1])

		var errList []error
		var numbersReceived int
		for numbers != nil || errs != nil {
			select {
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				errList = append(errList, err)
				if len(errList) == 3 {
					cancel()
					time.Sleep(100 * time.Microsecond) // block the main thread to avoid context not being canceled
				}
			case _, ok := <-numbers:
				if !ok {
					numbers = nil
					continue
				}
				numbersReceived++
			}
		}

		assert.Len(t, errList, 3)
		assert.Equal(t, numbersReceived, counts[0][0]+counts[1][0])
		assert.LessOrEqual(t, 1, counts[0][1])
		assert.LessOrEqual(t, 1, counts[1][1])
	})

	t.Run("without error channel", func(t *testing.T) {
		numbers := make(chan int)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			wg.Wait()
			close(numbers)
		}()

		var counts [2]int
		go func(count *int) {
			var err error
			var errCount int
			*count, errCount, err = ForwardWithCounts(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC := make(chan int)
				go func() {
					for i := 0; i < 50; i++ {
						intC <- i
					}
					close(intC)
				}()
				return intC, nil
			}, numbers, nil)
			assert.NoError(t, err)
			assert.Zero(t, errCount)
			wg.Done()
		}(&counts[0])

		go func(count *int) {
			var err error
			var errCount int
			*count, errCount, err = ForwardWithCounts(context.TODO(), func(context.Context) (<-chan int, <-chan error) {
				intC := make(chan int)
				go func() {
					for i := 50; i < 100; i++ {
						intC <- i
					}
					close(intC)
				}()
				return intC, nil
			}, numbers, nil)
			assert.NoError(t, err)
			assert.Zero(t, errCount)
			wg.Done()
		}(&counts[1])

		var ints []int
		for numbers != nil {
			j, ok := <-numbers
			if !ok {
				numbers = nil
				continue
			}
			ints = append(ints, j)
		}

		ref := make([]int, 100)
		for x := 0; x < 100; x++ {
			ref[x] = x
		}

		assert.ElementsMatch(t, ref, ints)
		assert.Equal(t, 50, counts[0])
		assert.Equal(t, 50, counts[1])
	})
}

func TestSend(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("no error", func(t *testing.T) {
		numbers := make(chan int)
		ctx := context.TODO()

		var lastErr error
		go func() {
			defer close(numbers)

			ints := make([]int, 100)
			for i := 0; i < 100; i++ {
				ints[i] = i
			}
			lastErr = Send(ctx, numbers, ints...)
		}()

		var nmr int
		for numbers != nil {
			if number, ok := <-numbers; !ok {
				numbers = nil
			} else {
				nmr = number
			}
		}

		assert.Equal(t, 99, nmr)
		assert.NoError(t, lastErr)
	})

	t.Run("context cancellation", func(t *testing.T) {
		numbers := make(chan int)
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		done := make(chan struct{})
		var lastErr error
		go func() {
			defer close(done)

			ints := make([]int, 100)
			for i := 0; i < 100; i++ {
				ints[i] = i
			}
			lastErr = Send(ctx, numbers, ints...)
		}()

		const breakAt = 23
		var nmr int
		func() {
			for {
				select {
				case <-done:
					return
				case nmr = <-numbers:
					if nmr == breakAt {
						cancel()
						// give the stack some time to propagate the cancellation.
						// Normally we would quit here either way
						time.Sleep(50 * time.Microsecond)
					}
				}
			}
		}()

		assert.LessOrEqual(t, breakAt, nmr)
		assert.Greater(t, 100, nmr)
		assert.ErrorIsf(t, lastErr, context.Canceled,
			"context was expected to be canceled, but is: %q", lastErr)
	})
}

func TestSendWithCount(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("no error", func(t *testing.T) {
		numbers := make(chan int)
		ctx := context.TODO()

		var lastErr error
		var sent int
		go func() {
			defer close(numbers)

			ints := make([]int, 100)
			for i := 0; i < 100; i++ {
				ints[i] = i
			}
			sent, lastErr = SendWithCount(ctx, numbers, ints...)
		}()

		var nmr int
		for numbers != nil {
			if number, ok := <-numbers; !ok {
				numbers = nil
			} else {
				nmr = number
			}
		}

		assert.Equal(t, 99, nmr)
		assert.NoError(t, lastErr)
		assert.Equal(t, 100, sent)
	})

	t.Run("context cancellation", func(t *testing.T) {
		numbers := make(chan int)
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		done := make(chan struct{})
		var lastErr error
		var sent int
		go func() {
			defer close(done)

			ints := make([]int, 100)
			for i := 0; i < 100; i++ {
				ints[i] = i
			}
			sent, lastErr = SendWithCount(ctx, numbers, ints...)
		}()

		const breakAt = 23
		var nmr int
	loop:
		for {
			select {
			case <-done:
				break loop
			case nmr = <-numbers:
				if nmr == breakAt {
					cancel()
					// give the stack some time to propagate the cancellation.
					// Normally we would quit here either way
					time.Sleep(50 * time.Microsecond)
				}
			}
		}

		assert.LessOrEqual(t, breakAt, nmr)
		assert.Greater(t, 100, sent)
		assert.ErrorIsf(t, lastErr, context.Canceled,
			"context was expected to be canceled, but is: %q", lastErr)
	})
}

func TestSendE(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("context cancellation", func(t *testing.T) {
		errors := make(chan error)
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		done := make(chan struct{})
		var ctxErr error
		go func() {
			defer close(done)

			errs := make([]error, 100)
			for i := 0; i < 100; i++ {
				if i%5 == 0 {
					// this cannot be done with Send directly due to type inference issue
					errs[i] = customError(fmt.Sprintf("error determining number %d", i))
					continue
				}
				errs[i] = fmt.Errorf("error determining number %d", i)
			}
			ctxErr = SendE(ctx, errors, errs...)
		}()

		var number int
		const breakAt = 25 // must be a multiple of 5
		var lastErr error
	loop:
		for {
			select {
			case <-done:
				break loop
			case lastErr = <-errors:
				if number == breakAt {
					cancel()
					// give the stack some time to propagate the cancellation.
					// Normally we would quit here either way
					time.Sleep(50 * time.Microsecond)
				}
				number++
			}
		}

		assert.IsType(t, customError(""), lastErr)
		assert.ErrorIs(t, ctxErr, context.Canceled)
	})
}

func TestSendEWithCount(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("context cancellation", func(t *testing.T) {
		errors := make(chan error)
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		done := make(chan struct{})
		var ctxErr error
		var errsSent int
		go func() {
			defer close(done)

			errs := make([]error, 100)
			for i := 0; i < 100; i++ {
				if i%5 == 0 {
					// this cannot be done with Send directly due to type inference issue
					errs[i] = customError(fmt.Sprintf("error determining number %d", i))
					continue
				}
				errs[i] = fmt.Errorf("error determining number %d", i)
			}
			errsSent, ctxErr = SendEWithCount(ctx, errors, errs...)
		}()

		var number int
		const breakAt = 25 // must be a multiple of 5
		var lastErr error
		func() {
			for {
				select {
				case <-done:
					return
				case lastErr = <-errors:
					if number == breakAt {
						cancel()
						// give the stack some time to propagate the cancellation.
						// Normally we would quit here either way
						time.Sleep(50 * time.Microsecond)
					}
					number++
				}
			}
		}()

		assert.IsType(t, customError(""), lastErr)
		assert.ErrorIsf(t, ctxErr, context.Canceled,
			"context was expected to be canceled, but is: %q", ctxErr)
		assert.LessOrEqual(t, errsSent, breakAt+1)
	})
}

func TestWrap(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("success with items discarded", func(t *testing.T) {
		src := make(chan int, 100)
		go func() {
			for i := 0; i < 100; i++ {
				src <- i
			}
			close(src)
		}()

		proc := func(_ context.Context, item int) (float64, bool) {
			if item%3 != 0 {
				return 0, false
			}
			return float64(item) / 3, true
		}

		var divBy3 []float64
		for nmbr := range Wrap(context.TODO(), src, proc) {
			divBy3 = append(divBy3, nmbr)
		}

		ref := make([]float64, 34)
		for j := 0; j < 34; j++ {
			ref[j] = float64(j)
		}
		assert.Equal(t, ref, divBy3)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		src := make(chan int)
		go func() {
			for i := 0; i < 100; i++ {
				Send(ctx, src, i)
			}
			close(src)
		}()

		var call int
		proc := func(_ context.Context, item int) (float64, bool) {
			if item%3 != 0 {
				if call++; call == 10 {
					cancel()
				}
				return 0, false
			}
			return float64(item) / 3, true
		}

		var divBy3 []float64
		for nmbr := range Wrap(ctx, src, proc) {
			divBy3 = append(divBy3, nmbr)
		}

		ref := make([]float64, 5)
		for j := 0; j < 5; j++ {
			ref[j] = float64(j)
		}
		assert.Equal(t, ref, divBy3)
	})

	t.Run("called with nil source", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Millisecond)
		defer cancel()

		var src chan int
		dst := Wrap(ctx, src, func(context.Context, int) (int, bool) {
			return 1, true
		})

		var items []int
		for nmbr := range dst {
			items = append(items, nmbr)
		}
		assert.Equal(t, ([]int)(nil), items)
		assert.NoError(t, ctx.Err())
	})
}

func TestWrapSpread(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("success with items discarded", func(t *testing.T) {
		src := make(chan int, 10)
		go func() {
			for i := 0; i < 10; i++ {
				src <- i
			}
			close(src)
		}()

		proc := func(_ context.Context, item int) []int64 {
			if item%3 == 0 {
				return nil
			}

			n := make([]int64, 10)
			for i := 0; i < 10; i++ {
				n[i] = int64(item*10 + i)
			}

			return n
		}

		var weirdNumbers []int64
		for nmbr := range WrapSpread(context.TODO(), src, proc) {
			weirdNumbers = append(weirdNumbers, nmbr)
		}

		ref := make([]int64, 0, 60)
		for i := 0; i < 10; i++ {
			if i%3 == 0 {
				continue
			}
			for j := 0; j < 10; j++ {
				ref = append(ref, int64(i*10+j))
			}
		}
		assert.Equal(t, ref, weirdNumbers)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		src := make(chan int, 10)
		go func() {
			for i := 0; i < 10; i++ {
				Send(ctx, src, i)
			}
			close(src)
		}()

		var call int
		proc := func(_ context.Context, item int) []int64 {
			if call++; call >= 5 {
				cancel()
				return nil
			}
			if item%3 != 0 {
				return nil
			}

			n := make([]int64, 10)
			for i := 0; i < 10; i++ {
				n[i] = int64(item*10 + i)
			}

			return n
		}

		var weirdNumbers []int64
		for nmbr := range WrapSpread(ctx, src, proc) {
			weirdNumbers = append(weirdNumbers, nmbr)
		}

		ref := make([]int64, 0, 20)
		for i := 0; i < 4; i += 3 {
			for j := 0; j < 10; j++ {
				ref = append(ref, int64(i*10+j))
			}
		}
		assert.Equal(t, ref, weirdNumbers)
	})

	t.Run("called with nil source", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Millisecond)
		defer cancel()

		var src chan int
		dst := WrapSpread(ctx, src, func(context.Context, int) []int {
			return []int{1, 2, 4}
		})

		var items []int
		for nmbr := range dst {
			items = append(items, nmbr)
		}
		assert.Equal(t, ([]int)(nil), items)
		assert.NoError(t, ctx.Err())
	})
}

type customError string

func (c customError) Error() string {
	return string(c)
}
