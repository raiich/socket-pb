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

func (m *CurrentState[S]) Set(next S) error {
	if m.timer != nil {
		_ = m.timer.Stop()
		m.timer = nil
	}
	m.current = next
	return nil
}

func (m *CurrentState[S]) SetAfter(dispatcher Dispatcher, d time.Duration, next func() S) error {
	if m.timer != nil {
		return fmt.Errorf("timer already set")
	}
	m.timer = dispatcher.AfterFunc(d, func() {
		m.timer = nil
		m.current = next()
	})
	return nil
}

type Dispatcher interface {
	AfterFunc(duration time.Duration, f func()) task.Timer
}
