package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/raiich/socket-pb/lib/errors"
	"github.com/raiich/socket-pb/lib/log"
	"github.com/raiich/socket-pb/lib/must"
	"github.com/raiich/socket-pb/room"
	"github.com/raiich/socket-pb/stream"
	"github.com/raiich/socket-pb/stream/tcp"
)

var logger = slog.New(&log.Handler{
	Handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}),
})

func main() {
	addr := ":8081"
	lis := must.Must(tcp.Listen(addr))
	defer func() {
		logOnError(lis.Close())
	}()

	logger.Info("listening", "addr", lis.Addr())

	ctx := context.TODO()
	settings := &room.ManagerSettings{
		Context:        ctx,
		DispatcherType: room.DispatcherTypeAsync,
	}
	rooms := room.NewManager(settings)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn := must.Must(lis.Accept())
		st := stream.New(conn)
		r := rooms.GetOrCreate("room-1")
		must.NoError(r.Dispatcher.InvokeSync(ctx, func() {
			client := &room.DirectClient{
				Stream:   st,
				ClientID: r.NextClientID(),
			}
			r.AddClient(client)
		}))

		for {
			buf := make([]byte, 1024)
			n, err := st.Read(buf)
			if errors.Is(err, io.EOF) {
				return
			}
			must.NoError(r.Dispatcher.InvokeSync(ctx, func() {
				for _, client := range r.Clients {
					must.Must(client.Write(buf[:n]))
				}
			}))
		}
	}()
	wg.Wait()
}

func logOnError(err error, args ...any) {
	if err != nil {
		logger.Error("unexpected error", append(args, "error", err)...)
	}
}
