package task

import (
	"sync/atomic"
	"time"
)

type Timer interface {
	Stop() bool
}

type taskTimer struct {
	finished atomic.Bool
	timer    *time.Timer
}

// tryFire executes Task if taskTimer.Stop is not called.
// If taskTimer.Stop is called immediately before time.AfterFunc is fired, Task will not be executed.
func (t *taskTimer) tryFire(task Task) {
	// finished == false => timer fire is valid => callable
	if t.finished.CompareAndSwap(false, true) {
		// Although such code is unlikely to be written in practice,
		// but `finished = true` is set before calling `f()`
		// to avoid the possibility that `timer.Stop()` is called inside `f()` and returns `true`.
		task.Exec()
	}
}

func (t *taskTimer) Stop() bool {
	if t.finished.CompareAndSwap(false, true) {
		_ = t.timer.Stop()
		return true
	}
	return false
}
