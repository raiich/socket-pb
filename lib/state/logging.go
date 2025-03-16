package state

import (
	"reflect"
	"time"

	"github.com/raiich/socket-pb/internal/log"
)

type LoggingCurrentState[S State] struct {
	base CurrentState[S]
}

func (m *LoggingCurrentState[S]) Get() S {
	return m.base.Get()
}

func (m *LoggingCurrentState[S]) Set(next S) {
	from := m.base.Get()
	log.Debug("state transition", "from", reflect.TypeOf(from), "to", reflect.TypeOf(next))
	m.base.Set(next)
}

func (m *LoggingCurrentState[S]) AfterFunc(dispatcher Dispatcher, d time.Duration, f func()) error {
	log.Debug("state AfterFunc", "duration", d)
	return m.base.AfterFunc(dispatcher, d, f)
}
