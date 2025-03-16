package task

import (
	"context"
	"sync/atomic"

	"github.com/raiich/socket-pb/lib/errors"
)

var (
	ErrClosedQueue = errors.Newf("closed queue")
)

type Queue[T any] struct {
	queue   chan T
	ctx     context.Context
	cancel  context.CancelCauseFunc
	closing atomic.Bool
}

func (w *Queue[T]) Enqueue(ctx context.Context, elem T) error {
	select {
	case <-w.ctx.Done():
		return context.Cause(w.ctx)
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		select {
		case w.queue <- elem:
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-w.ctx.Done():
			return context.Cause(w.ctx)
		}
	}
}

func (w *Queue[T]) Close() error {
	return w.closeByErr(ErrClosedQueue)
}

func (w *Queue[T]) closeByErr(reason error) error {
	if !w.closing.CompareAndSwap(false, true) {
		err := context.Cause(w.ctx)
		return errors.Wrapf(err, "queue already closed")
	}
	w.cancel(reason)
	return nil

	// avoid close(queue) here,
	// for the case "send to closed channel" error. GC will collect queue.
}

// Dequeue returns the next element in the queue.
// If the queue is empty, it will block until an element is available.
// If Queue.Close is called, it will return ErrClosedQueue as error value.
func (w *Queue[T]) Dequeue() (T, error) {
	select {
	case <-w.ctx.Done():
		var zero T
		return zero, context.Cause(w.ctx)
	default:
		select {
		case <-w.ctx.Done():
			var zero T
			return zero, context.Cause(w.ctx)
		case elem := <-w.queue:
			return elem, nil
		}
	}
}

func NewQueue[T any](size int) *Queue[T] {
	ctx, cancel := context.WithCancelCause(context.Background())
	return &Queue[T]{
		queue:  make(chan T, size),
		ctx:    ctx,
		cancel: cancel,
	}
}
