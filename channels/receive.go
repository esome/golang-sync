// Package channels provides a set of utilities for working with channels.

package channels

import (
	"context"
	"sync"
)

// ChunkAndDo reads the given channel up until X elements, defined by size parameter, have been consumed,
// and then calls the doer function provided, passing the actual chunk of data.
//
// The function is context aware.
func ChunkAndDo[T any](ctx context.Context, src <-chan T, size uint, doer func(context.Context, []T) error) error {
	if size == 0 {
		return ErrInvalidChunkSize
	}

	iSize := int(size)
	items := make([]T, 0, iSize)
	for src != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-src:
			if !ok {
				src = nil
				continue
			}
			items = append(items, item)
			if len(items) == iSize {
				if err := doer(ctx, items); err != nil {
					return err
				}
				items = items[:0]
			}
		}
	}

	if len(items) > 0 {
		return doer(ctx, items)
	}

	return ctx.Err()
}

// Collect reads the given channel and returns a slice with all elements received on the src channel.
//
// The function is context aware.
func Collect[T any](ctx context.Context, src <-chan T) ([]T, error) {
	items := make([]T, 0, cap(src))
	for src != nil {
		select {
		case <-ctx.Done():
			return items, ctx.Err()
		case item, ok := <-src:
			if !ok {
				src = nil
				continue
			}
			items = append(items, item)
		}
	}
	return items, ctx.Err()
}

// Drain will drain the provided src channel up until all elements have been drained and the channel has been closed,
// or the given context has been canceled.
//
// If you need to drain the channel in any case, pass a context not being canceled.
//
// ⚠️ Beware: The function will block until the src channel is closed then.
func Drain[T any](ctx context.Context, src <-chan T) error {
	_, err := DrainWithCount(ctx, src)
	return err
}

// DrainWithCount works like [Drain], but additionally returns the number of items discarded.
func DrainWithCount[T any](ctx context.Context, src <-chan T) (int, error) {
	var items int

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case _, ok := <-src:
			if !ok {
				break loop
			}
			items++
		}
	}
	return items, ctx.Err()
}

// Merge joins the provided list of source channels into a single one,
// which can then be consumed in the main thread
//
// It returns the joined T channel and takes care of closing it properly
// after having consumed all the source channels until they are closed, or the context is canceled.
// In case of context cancellation, make sure to cancel the producing channels to avoid leaking goroutines.
func Merge[T any](ctx context.Context, sources ...<-chan T) <-chan T {
	merged := make(chan T, len(sources))
	var wg sync.WaitGroup
	wg.Add(len(sources))

	for _, src := range sources {
		go func(src <-chan T) {
			defer wg.Done()
			for src != nil {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-src:
					if !ok {
						return
					}
					_ = Send(ctx, merged, item)
				}
			}
		}(src)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}
