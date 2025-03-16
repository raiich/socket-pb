package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/raiich/socket-pb/lib/log"
	"github.com/raiich/socket-pb/lib/task"
	"github.com/raiich/socket-pb/stream"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
	"github.com/raiich/socket-pb/stream/tcp"
)

var logger = slog.New(&log.Handler{
	Handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}),
})

func main() {
	addr := ":8081"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	logger.Info("listening", "addr", lis.Addr())
	newApp := func(ss stream.ServerStream, hs *stream.ClientHandshake) (stream.ServerHandler, error) {
		dispatcher := task.NewAsyncDispatcher(context.TODO())
		go func() { logOnError(dispatcher.Run()) }()
		handshake := &stream.ServerHandshake{
			Status: packet.HandshakeStatus_OK,
		}
		if err := ss.WriteServerHandshake(context.TODO(), handshake); err != nil {
			return nil, err
		}
		payload := &stream.Payload{
			Payload: []byte("hi"),
		}
		if err := ss.WritePayload(context.TODO(), payload); err != nil {

		}
		return &EchoApp{
			Stream:     ss,
			Listener:   lis,
			dispatcher: dispatcher,
		}, nil
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		sv := tcp.NewServer(newApp)
		if err := sv.Serve(lis); err != nil {
			logger.Error("failed to serve", "error", err)
		}
	}()
	wg.Wait()
}

type EchoApp struct {
	Stream     stream.ServerStream
	Listener   io.Closer
	dispatcher *task.AsyncDispatcher
}

func (a *EchoApp) Dispatcher() stream.Dispatcher {
	return a.dispatcher
}

func (a *EchoApp) OnPayload(payload *stream.Payload) {
	logger.Info("received message", "payload", string(payload.Payload))
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	if err := a.Stream.WritePayload(ctx, payload); err != nil {
		logger.Warn("failed to write payload", "error", err)
		a.Stream.StartShutdown(ctx)
	}
}

func (a *EchoApp) OnClose(reason *stream.Close) {
	logger.Debug("received ConnectionClose", "message", reason)
	a.dispatcher.AfterFunc(10*time.Millisecond, func() {
		logOnError(a.dispatcher.Stop())
	})
}

func logOnError(err error, args ...any) {
	if err != nil {
		logger.Error("unexpected error", append(args, "error", err)...)
	}
}
