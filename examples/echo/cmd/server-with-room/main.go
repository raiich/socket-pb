package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/raiich/socket-pb/lib/log"
	"github.com/raiich/socket-pb/room"
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
	ctx := context.TODO()
	addr := ":8081"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	logger.Info("listening", "addr", lis.Addr())

	wg := sync.WaitGroup{}
	wg.Add(1)
	settings := &room.ManagerSettings{
		Context:        ctx,
		DispatcherType: room.DispatcherTypeAsync,
	}
	rooms := room.NewManager(settings)
	go func() {
		defer wg.Done()
		sv := tcp.NewServer(func(ss stream.ServerStream, hs *stream.ClientHandshake) (stream.ServerHandler, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var client room.Client
			r := rooms.GetOrCreate("room-1")
			err := r.Dispatcher.InvokeSync(ctx, func() {
				if r.Handler == nil {
					r.Handler = &ChatRoom{
						Stream:   ss,
						Listener: lis,
						base:     r,
					}
				}

				client = &room.DirectClient{
					ServerStream: ss,
					ClientID:     r.NextClientID(),
					ClaimMap: map[string]any{
						"name": hs.Params["name"],
					},
				}
				r.AddClient(client)
			})
			if err != nil {
				//TODO implement me
				panic("implement me")
			}

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
				panic("implement me")
			}
			return &StreamRoomAdapter{
				ClientID: client.ID(),
				Stream:   ss,
				Listener: lis,
				Room:     r,
			}, nil

		})
		if err := sv.Serve(lis); err != nil {
			logger.Error("failed to serve", "error", err)
		}
	}()
	wg.Wait()
}

type StreamRoomAdapter struct {
	ClientID room.ClientID
	Stream   stream.ServerStream
	Listener io.Closer
	Room     *room.Room
}

func (a *StreamRoomAdapter) Dispatcher() stream.Dispatcher {
	return a.Room.Dispatcher
}

func (a *StreamRoomAdapter) OnPayload(payload *stream.Payload) {
	a.Room.Handler.OnPayload(a.ClientID, payload)
}

func (a *StreamRoomAdapter) OnClose(reason *stream.Close) {
	logger.Debug("received stream.Close", "message", reason)
}

type ChatRoom struct {
	Stream   stream.ServerStream
	Listener io.Closer
	base     *room.Room
}

func (a *ChatRoom) OnPayload(clientID room.ClientID, payload *stream.Payload) {
	ctx, cancel := context.WithTimeout(a.base.Dispatcher.Context(), 5*time.Second)
	defer cancel()
	from := a.base.Clients[clientID]
	for _, to := range a.base.Clients {
		fromID, toID := from.ID(), to.ID()
		fromName, toName := from.Claims()["name"], to.Claims()["name"]
		prefix := fmt.Sprintf("%d %v->%d %v: ", fromID, fromName, toID, toName)
		logOnError(to.WritePayload(ctx, &stream.Payload{
			Payload: append([]byte(prefix), payload.Payload...),
		}))
	}
}

func logOnError(err error, args ...any) {
	if err != nil {
		logger.Error("unexpected error", append(args, "error", err)...)
	}
}
