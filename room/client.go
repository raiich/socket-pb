package room

import (
	"context"

	"github.com/raiich/socket-pb/stream"
)

type ClientID int

type Client interface {
	ID() ClientID
	Claims() map[string]any
	WritePayload(ctx context.Context, msg *stream.Payload) error
	StartShutdown(ctx context.Context)
}

type DirectClient struct {
	stream.ServerStream
	ClientID ClientID
	ClaimMap map[string]any
}

func (c *DirectClient) ID() ClientID {
	return c.ClientID
}

func (c *DirectClient) Claims() map[string]any {
	return c.ClaimMap
}
