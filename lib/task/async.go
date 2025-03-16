package task

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/raiich/socket-pb/lib/errors"
)

const (
	defaultEnqueueTimeout = 8 * time.Second
	defaultTaskQueueSize  = 256
)

// AsyncDispatcher executes Task sequentially in Run loop.
type AsyncDispatcher struct {
	queue   Queue[Task]
	started atomic.Bool
}

func (d *AsyncDispatcher) Context() context.Context {
	return d.queue.ctx
}

func (d *AsyncDispatcher) Launch() error {
	return d.Run()
}

// Run execute Task in Run loop.
// Call Stop or StopByError to exit Run.
func (d *AsyncDispatcher) Run() error {
	if !d.started.CompareAndSwap(false, true) { // avoid multiple start
		return fmt.Errorf("dispatcher already started")
	}

	defer func() { _ = d.StopByError(fmt.Errorf("dispatcher is finished")) }()
	for {
		task, err := d.queue.Dequeue()
		if err != nil {
			if errors.Is(err, ErrClosedQueue) {
				// ErrClosedQueue is ok
				return nil
			}
			return err
		}
		task.Exec()
	}
}

func (d *AsyncDispatcher) TaskQueue() *Queue[Task] {
	return &d.queue
}

// Stop is used to stop AsyncDispatcher immediately.
// Call Stop in AsyncDispatcher.InvokeFunc if you want to execute Task already enqueued in TaskQueue.
func (d *AsyncDispatcher) Stop() error {
	return d.queue.Close()
}

// StopByError is used to stop AsyncDispatcher immediately with error.
// You can get error reason in Run method return value.
// Call StopByError in AsyncDispatcher.InvokeFunc if you want to execute Task already enqueued in TaskQueue.
func (d *AsyncDispatcher) StopByError(reason error) error {
	return d.queue.closeByErr(reason)
}

// AfterFunc enqueues f to TaskQueue after specified duration.
func (d *AsyncDispatcher) AfterFunc(duration time.Duration, f func()) Timer {
	timer := &taskTimer{}
	timer.timer = time.AfterFunc(duration, func() {
		ctx, cancel := context.WithTimeout(d.queue.ctx, defaultEnqueueTimeout)
		defer cancel()

		// enqueue task to execute f in same goroutine with Run loop
		d.InvokeFunc(ctx, func() {
			timer.tryFire(f)
		})
	})
	return timer
}

// InvokeFunc enqueues f to TaskQueue.
func (d *AsyncDispatcher) InvokeFunc(ctx context.Context, f func()) {
	err := d.queue.Enqueue(ctx, f)
	if err != nil {
		_ = d.StopByError(err)
	}
}

func (d *AsyncDispatcher) InvokeSync(ctx context.Context, f func()) error {
	done := make(chan struct{})
	d.InvokeFunc(ctx, func() {
		defer close(done)
		f()
	})
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-done:
		return nil
	}
}

func NewAsyncDispatcher(parent context.Context) *AsyncDispatcher {
	ctx, cancel := context.WithCancelCause(parent)
	return &AsyncDispatcher{
		queue: Queue[Task]{
			queue:  make(chan Task, defaultTaskQueueSize),
			ctx:    ctx,
			cancel: cancel,
		},
	}
}
