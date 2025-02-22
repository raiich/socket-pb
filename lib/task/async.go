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

// Run execute Task in Run loop.
// Call Stop or StopByError to exit Run.
func (d *AsyncDispatcher) Run(ctx context.Context) error {
	if !d.started.CompareAndSwap(false, true) { // avoid multiple start
		return fmt.Errorf("dispatcher already started")
	}

	defer func() { _ = d.StopByError(fmt.Errorf("dispatcher is finished")) }()
	for {
		task, err := d.queue.Dequeue(ctx)
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

// InvokeFunc enqueues f to TaskQueue.
func (d *AsyncDispatcher) InvokeFunc(ctx context.Context, f func()) {
	err := d.queue.Enqueue(ctx, f)
	if err != nil {
		_ = d.StopByError(err)
	}
}

// AfterFunc enqueues f to TaskQueue after specified duration.
func (d *AsyncDispatcher) AfterFunc(duration time.Duration, f func()) Timer {
	timer := &noMutexTimer{}
	timer.timer = time.AfterFunc(duration, func() {
		ctx, cancel := context.WithTimeout(d.queue.ctx, defaultEnqueueTimeout)
		defer cancel()

		// enqueue task to execute f in same goroutine with RunLoop
		d.InvokeFunc(ctx, func() {
			timer.tryFire(f)
		})
	})
	return timer
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

type noMutexTimer struct {
	// finished is updated and referenced in the goroutine of AsyncDispatcher.Run, so no mutex is used.
	finished bool
	timer    *time.Timer
}

// tryFire executes Task if noMutexTimer.Stop is not called.
// If noMutexTimer.Stop is called immediately before time.AfterFunc is fired, Task will not be executed.
func (t *noMutexTimer) tryFire(task Task) {
	if !t.finished {
		// !finished => timer fire is valid => callable

		// Although such code is unlikely to be written in practice,
		// but `finished = true` is set before calling `f()`
		// to avoid the possibility that `timer.Stop()` is called inside `f()` and returns `true`.
		t.finished = true
		task.Exec()
	}
}

func (t *noMutexTimer) Stop() bool {
	if t.finished {
		return false
	}
	t.finished = true
	_ = t.timer.Stop()
	return true
}
