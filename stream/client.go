package stream

import "context"

type NewClientHandler func(cs ClientStream, hs *ServerHandshake) (ClientHandler, error)

type ClientHandshake struct {
	Params map[string][]byte
}

type ClientStream interface {
	WritePayload(ctx context.Context, payload *Payload) error
	StartShutdown(ctx context.Context)
}

type ClientHandler interface {
	OnPayload(payload *Payload)
	OnClose(reason *Close)
}
