package state

import (
	"fmt"
	"time"

	"github.com/raiich/socket-pb/lib/task"
)

type State interface {
}

type CurrentState[S State] struct {
	current S
	timer   task.Timer
}

func (m *CurrentState[S]) Get() S {
	return m.current
}

func (m *CurrentState[S]) Set(next S) {
	if m.timer != nil {
		_ = m.timer.Stop()
		m.timer = nil
	}
	m.current = next
}

func (m *CurrentState[S]) AfterFunc(dispatcher Dispatcher, d time.Duration, f func()) error {
	if m.timer != nil {
		return fmt.Errorf("timer already set")
	}
	m.timer = dispatcher.AfterFunc(d, func() {
		m.timer = nil
		f()
	})
	return nil
}

type Dispatcher interface {
	AfterFunc(duration time.Duration, f func()) task.Timer
}
