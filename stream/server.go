package stream

import (
	"context"

	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
)

type NewServerHandler func(ss ServerStream, hs *ClientHandshake) (ServerHandler, error)

type ServerHandshake struct {
	Status packet.HandshakeStatus
	Params map[string][]byte
}

type ServerStream interface {
	WriteServerHandshake(ctx context.Context, handshake *ServerHandshake) error
	WritePayload(ctx context.Context, payload *Payload) error
	StartShutdown(ctx context.Context)
}

type ServerHandler interface {
	Dispatcher() Dispatcher
	OnPayload(payload *Payload)
	OnClose(reason *Close)
}
