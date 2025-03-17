package room

import (
	"context"

	"github.com/raiich/socket-pb/stream"
)

type ClientID int

type Client interface {
	ID() ClientID
	Claims() map[string]any
	Write(p []byte) (n int, err error)
	StartShutdown(ctx context.Context)
}

type DirectClient struct {
	*stream.Stream
	ClientID ClientID
	ClaimMap map[string]any
}

func (c *DirectClient) ID() ClientID {
	return c.ClientID
}

func (c *DirectClient) Claims() map[string]any {
	return c.ClaimMap
}

func (c *DirectClient) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}
