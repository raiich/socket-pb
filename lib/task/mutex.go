package task

import (
	"sync"
	"time"
)

// MutexDispatcher executes Task in same goroutine of caller with sync.Mutex.
// MutexDispatcher is used to execute Task sequentially in same goroutine of caller, using sync.Mutex.
type MutexDispatcher struct {
	mu sync.Mutex
}

// InvokeFunc executes f in same goroutine of caller, using sync.Mutex.
func (d *MutexDispatcher) InvokeFunc(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	f()
}

// AfterFunc executes f in same goroutine of caller after specified duration, using sync.Mutex.
func (d *MutexDispatcher) AfterFunc(duration time.Duration, f func()) Timer {
	timer := &mutexTimer{
		mu: &d.mu,
	}
	timer.timer = time.AfterFunc(duration, func() {
		timer.tryFire(f)
	})
	return timer
}

func NewMutexDispatcher() *MutexDispatcher {
	return &MutexDispatcher{}
}

type mutexTimer struct {
	mu       *sync.Mutex
	finished bool
	timer    *time.Timer
}

// tryFire executes Task if mutexTimer.Stop is not called.
// If mutexTimer.Stop is called immediately before time.AfterFunc is fired, Task will not be executed.
func (t *mutexTimer) tryFire(task Task) {
	// to execute the task sequentially after invoked by AfterFunc
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.finished {
		// !finished => timer fire is valid => callable
		t.finished = true
		task.Exec()
	}
}

func (t *mutexTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.finished {
		return false
	}
	t.finished = true
	_ = t.timer.Stop()
	return true
}
