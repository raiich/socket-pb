package main

import (
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/raiich/socket-pb/lib/log"
	"github.com/raiich/socket-pb/lib/must"
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
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn := must.Must(lis.Accept())
		handle(conn)
	}()
	wg.Wait()
}

func handle(conn *tcp.Conn) {
	must.Must(io.Copy(conn, conn))
}

func logOnError(err error, args ...any) {
	if err != nil {
		logger.Error("unexpected error", append(args, "error", err)...)
	}
}
