package task

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raiich/socket-pb/lib/errors"
)

var errStopped = errors.Newf("dispatcher stopped")

type MutexDispatcher struct {
	base    *MutexSimpleDispatcher
	ctx     context.Context
	cancel  context.CancelCauseFunc
	closing atomic.Bool
}

func (d *MutexDispatcher) Context() context.Context {
	return d.ctx
}

func (d *MutexDispatcher) AfterFunc(duration time.Duration, f func()) Timer {
	return d.base.AfterFunc(duration, f)
}

func (d *MutexDispatcher) InvokeFunc(_ context.Context, f func()) {
	d.base.InvokeFunc(f)
}

func (d *MutexDispatcher) InvokeSync(_ context.Context, f func()) error {
	d.base.InvokeFunc(f)
	return nil
}

func (d *MutexDispatcher) Launch() error {
	select {
	case <-d.ctx.Done():
		err := context.Cause(d.ctx)
		if err != nil && !errors.Is(err, errStopped) {
			return err
		}
		return nil
	}
}

func (d *MutexDispatcher) Stop() error {
	return d.StopByError(errStopped)
}

func (d *MutexDispatcher) StopByError(reason error) error {
	if !d.closing.CompareAndSwap(false, true) {
		err := context.Cause(d.ctx)
		return errors.Wrapf(err, "already stopped")
	}
	d.cancel(reason)
	return nil
}

func NewMutexDispatcher(parent context.Context) *MutexDispatcher {
	ctx, cancel := context.WithCancelCause(parent)
	return &MutexDispatcher{
		base:   NewMutexSimpleDispatcher(),
		ctx:    ctx,
		cancel: cancel,
	}
}

// MutexSimpleDispatcher executes Task in same goroutine of caller with sync.Mutex.
// MutexSimpleDispatcher is used to execute Task sequentially in same goroutine of caller, using sync.Mutex.
type MutexSimpleDispatcher struct {
	mu sync.Mutex
}

// InvokeFunc executes f in same goroutine of caller, using sync.Mutex.
func (d *MutexSimpleDispatcher) InvokeFunc(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	f()
}

// AfterFunc executes f in same goroutine of caller after specified duration, using sync.Mutex.
func (d *MutexSimpleDispatcher) AfterFunc(duration time.Duration, f func()) Timer {
	timer := &taskTimer{}
	timer.timer = time.AfterFunc(duration, func() {
		d.InvokeFunc(func() {
			timer.tryFire(f)
		})
	})
	return timer
}

func NewMutexSimpleDispatcher() *MutexSimpleDispatcher {
	return &MutexSimpleDispatcher{}
}
