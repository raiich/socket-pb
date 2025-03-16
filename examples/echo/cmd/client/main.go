package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/raiich/socket-pb/lib/log"
	"github.com/raiich/socket-pb/lib/task"
	"github.com/raiich/socket-pb/stream"
	"github.com/raiich/socket-pb/stream/tcp"
)

var logger = slog.New(&log.Handler{
	Handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}),
})

func main() {
	ctx, cancel := context.WithCancelCause(context.TODO())
	defer cancel(context.Canceled)

	var dispatcher Dispatcher
	switch os.Getenv("DISPATCHER") {
	case "async":
		dispatcher = task.NewAsyncDispatcher(ctx)
	case "mutex":
		dispatcher = task.NewMutexDispatcher(ctx)
	default:
		dispatcher = task.NewMutexDispatcher(ctx)
	}

	appRoot := &AppRoot{
		Dispatcher: dispatcher,
	}
	tcpAddr := "localhost:8081"
	err := tcp.Dial(dispatcher, tcpAddr, appRoot.NewTCPApp)
	if err != nil {
		logger.Error("failed to start tcp", "addr", tcpAddr, "error", err)
		return
	}
	logger.Info("starting", "addr", tcpAddr)
	if err := dispatcher.Launch(); err != nil {
		panic("implement me")
	}
}

type AppRoot struct {
	Dispatcher Dispatcher
}

func (r *AppRoot) NewTCPApp(cs stream.ClientStream, handshake *stream.ServerHandshake) (stream.ClientHandler, error) {
	ctx, cancel := context.WithTimeout(r.Dispatcher.Context(), 5*time.Second)
	defer cancel()

	logger.Info("connected")
	a := &EchoApp{
		Stream: cs,
		root:   r,
	}
	payload := &stream.Payload{
		Payload: []byte("hello"),
	}
	if err := a.Stream.WritePayload(ctx, payload); err != nil {
		return nil, err
	}
	a.sent++
	return a, nil
}

func (r *AppRoot) DisposeTCP() {
	// using AfterFunc for async execution
	r.Dispatcher.AfterFunc(0, func() {
		logOnError(r.Dispatcher.Stop())
	})
}

type EchoApp struct {
	Stream   stream.ClientStream
	root     *AppRoot
	sent     int
	received int
}

func (a *EchoApp) OnPayload(payload *stream.Payload) {
	ctx, cancel := context.WithTimeout(a.root.Dispatcher.Context(), 5*time.Second)
	defer cancel()

	logger.Info("received message", "payload", string(payload.Payload))
	a.received++
	if a.received > 3 {
		a.Stream.StartShutdown(ctx)
		logger.Debug("connection closed by me")
	}
	if a.sent < 3 {
		res := &stream.Payload{
			Payload: []byte(fmt.Sprintf("hello %v", a.sent)),
		}
		if err := a.Stream.WritePayload(ctx, res); err != nil {
			logger.Warn("failed to write payload", "error", err)
			panic(err)
		}
		a.sent++
		return
	}
}

func (a *EchoApp) OnClose(reason *stream.Close) {
	a.root.DisposeTCP()
	logger.Debug("closed stream", "message", reason)
}

func logOnError(err error, args ...any) {
	if err != nil {
		logger.Error("unexpected error", append(args, "error", err)...)
	}
}

type Dispatcher interface {
	Context() context.Context
	InvokeFunc(ctx context.Context, f func())
	AfterFunc(duration time.Duration, f func()) task.Timer
	Launch() error
	Stop() error
	StopByError(reason error) error
}
