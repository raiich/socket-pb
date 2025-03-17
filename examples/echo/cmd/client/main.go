package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/raiich/socket-pb/lib/log"
	"github.com/raiich/socket-pb/lib/must"
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

	go func() {
		tcpAddr := "localhost:8081"
		conn, err := tcp.Dial(tcpAddr)
		if err != nil {
			logger.Error("failed to start tcp", "addr", tcpAddr, "error", err)
			return
		}
		st := stream.New(conn)

		{
			payload := []byte("hello 1")
			must.Must(st.Write(payload))
			buf := make([]byte, len(payload))
			must.Must(io.ReadFull(st, buf))
			if !bytes.Equal(buf, payload) {
				panic("implement me")
			}
		}
		{
			payload := []byte("hello 2")
			must.Must(st.Write(payload))
			buf := make([]byte, len(payload))
			must.Must(io.ReadFull(st, buf))
			if !bytes.Equal(buf, payload) {
				panic("implement me")
			}
		}
		{
			payload := []byte("hello 3")
			must.Must(st.Write(payload))
			buf := make([]byte, len(payload))
			must.Must(io.ReadFull(st, buf))
			if !bytes.Equal(buf, payload) {
				panic("implement me")
			}
		}
		must.NoError(conn.Close())
		must.NoError(dispatcher.Stop())
	}()
	if err := dispatcher.Launch(); err != nil {
		panic("implement me")
	}
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
