// Package channels provides a set of utilities for working with channels.

package channels

import (
	"context"
	"slices"
	"sync"
)

// Broadcast consumes the src channel and yields the incoming messages on every receiver channel.
// It waits in each cycle until the message could be sent on every dst channel.
//
// The function is context aware.
func Broadcast[T any](ctx context.Context, src <-chan T, receiver ...chan<- T) error {
	_, err := BroadcastWithCount(ctx, src, receiver...)
	return err
}

// BroadcastWithCount works like [Broadcast], but additionally returns the number of items sent to all receivers.
// That doesn't mean that one or the other receiver might have received more messages.
//
// The function is context aware.
func BroadcastWithCount[T any](ctx context.Context, src <-chan T, receiver ...chan<- T) (int, error) {
	rcv := len(receiver)
	if rcv == 0 {
		return 0, ErrBroadcastWithNoReceivers
	}

	var items int
	for src != nil {
		select {
		case <-ctx.Done():
			return items, ctx.Err()
		case msg, ok := <-src:
			if !ok {
				src = nil
				continue
			}
			var wg sync.WaitGroup
			wg.Add(rcv)
			sent := make([]int, rcv)
			for i, dst := range receiver {
				go func(index int, m T, ch chan<- T) {
					sent[index], _ = SendWithCount(ctx, ch, m)
					wg.Done()
				}(i, msg, dst)
			}
			wg.Wait()
			items += slices.Min(sent)
		}
	}

	return items, ctx.Err()
}

type generateFN[T any] func(context.Context) (<-chan T, <-chan error)

// Forward pipes the data emitted on the channels returned by the generate function to the given outer channels
// without any other manipulation. It blocks until both channels coming from generate function have been closed
func Forward[T any](ctx context.Context, generate generateFN[T], items chan<- T, errs chan<- error) error {
	_, _, err := ForwardWithCounts(ctx, generate, items, errs)
	return err
}

// ForwardWithCounts works like [Forward], but additionally returns the number of items and errors forwarded.
func ForwardWithCounts[T any](
	ctx context.Context,
	generate generateFN[T],
	items chan<- T,
	errs chan<- error,
) (itemCount, errCount int, _ error) {
	itemsLocal, errsLocal := generate(ctx)
	for itemsLocal != nil || errsLocal != nil {
		select {
		case <-ctx.Done():
			return itemCount, errCount, ctx.Err()
		case err, ok := <-errsLocal:
			if !ok {
				errsLocal = nil
				continue
			}
			ec, _ := SendEWithCount(ctx, errs, err)
			errCount += ec
		case item, ok := <-itemsLocal:
			if !ok {
				itemsLocal = nil
				continue
			}
			ic, _ := SendWithCount(ctx, items, item)
			itemCount += ic
		}
	}
	return itemCount, errCount, ctx.Err()
}

// Send securely sends all items provided on channel dst in a context aware fashion.
//
// It returns an error in case the context was canceled, nil otherwise.
//
// Therefore, it's a shorthand notation for the following select statement
//
//	select {
//	case <-ctx.Done():
//		// abort operation
//	case dst <- item:
//	}
func Send[T any](ctx context.Context, dst chan<- T, items ...T) error {
	_, err := SendWithCount(ctx, dst, items...)
	return err
}

// SendWithCount works like [Send], but additionally returns the number of items sent.
func SendWithCount[T any](ctx context.Context, dst chan<- T, items ...T) (int, error) {
	var sent int
	for _, item := range items {
		select {
		case <-ctx.Done():
			return sent, ctx.Err()
		case dst <- item:
			sent++
		}
	}
	return sent, ctx.Err()
}

// SendE is an error type bound variant of [Send], to not run into type inference issues when using error types directly
func SendE(ctx context.Context, dst chan<- error, err ...error) error {
	return Send(ctx, dst, err...)
}

// SendEWithCount is an error type bound variant of [SendWithCount], to not run into type inference issues when using error types directly
func SendEWithCount(ctx context.Context, dst chan<- error, err ...error) (int, error) {
	return SendWithCount(ctx, dst, err...)
}

// Wrap consumes the src channel of type I passing every item received to the proc function provided, and yields
// the return value of it on the result channel of type O returned. The process function must return a boolean flag,
// whether the returned value is ok, and shall be pushed on the outgoing channel. If the ok flag is set to false,
// the item will be discarded.
//
// This outgoing channel is closed, once either the incoming channel has been closed, or the context has been canceled
func Wrap[I, O any](ctx context.Context, src <-chan I, proc func(context.Context, I) (O, bool)) <-chan O {
	out := make(chan O)

	go func() {
		defer close(out)

		for src != nil {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-src:
				if !ok {
					return
				}

				if o, ok := proc(ctx, item); ok {
					Send(ctx, out, o)
				}
			}
		}
	}()

	return out
}

// WrapSpread consumes the src channel of type I passing every item received to the proc function provided,
// and yields the return values of it on the result channel of type O returned. The typical slice semantics apply here.
// So if you wish the result to be ignored, just return a nil/empty slice from the proc function.
//
// This outgoing channel is closed, once either the incoming channel has been closed, or the context has been canceled
func WrapSpread[I, O any](ctx context.Context, src <-chan I, proc func(context.Context, I) []O) <-chan O {
	out := make(chan O)

	go func() {
		defer close(out)

		for src != nil {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-src:
				if !ok {
					return
				}

				Send(ctx, out, proc(ctx, item)...)
			}
		}
	}()

	return out
}
