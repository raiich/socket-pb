package stream

import (
	"context"
	"time"

	"github.com/raiich/socket-pb/lib/task"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
)

type Payload struct {
	Payload []byte
}

type Close struct {
	Reason packet.CloseReason
}

type Dispatcher interface {
	Context() context.Context
	InvokeFunc(ctx context.Context, f func())
	AfterFunc(duration time.Duration, f func()) task.Timer
}
