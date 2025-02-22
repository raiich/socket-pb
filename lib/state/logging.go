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

func (m *LoggingCurrentState[S]) Set(next S) error {
	from := m.base.Get()
	log.Info("state transition", "from", reflect.TypeOf(from), "to", reflect.TypeOf(next))
	return m.base.Set(next)
}

func (m *LoggingCurrentState[S]) SetAfter(dispatcher Dispatcher, d time.Duration, next func() S) error {
	return m.base.SetAfter(dispatcher, d, func() S {
		from := m.base.Get()
		to := next()
		log.Info("state transition", "from", reflect.TypeOf(from), "to", reflect.TypeOf(to))
		return to
	})
}
