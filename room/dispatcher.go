package room

import (
	"context"
	"time"

	"github.com/raiich/socket-pb/lib/task"
)

type DispatcherType int

const (
	DispatcherTypeAsync DispatcherType = iota
	DispatcherTypeMutex
)

type Dispatcher interface {
	Context() context.Context
	AfterFunc(duration time.Duration, f func()) task.Timer
	InvokeFunc(ctx context.Context, f func())
	InvokeSync(ctx context.Context, f func()) error
	Launch() error
	Stop() error
	StopByError(reason error) error
}
