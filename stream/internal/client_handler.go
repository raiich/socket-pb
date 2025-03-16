package internal

import (
	"context"
	"net"
	"time"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/stream"
)

type ClientHandler struct {
	dispatcher interface {
		Context() context.Context
		InvokeFunc(ctx context.Context, f func())
	}
	handler clientLoggingHandler
}

func (s *ClientHandler) invokeFunc(f func()) {
	ctx, cancel := context.WithTimeout(s.dispatcher.Context(), 5*time.Second)
	defer cancel()
	s.dispatcher.InvokeFunc(ctx, f)
}

func (s *ClientHandler) OnPayload(payload *stream.Payload) {
	s.invokeFunc(func() {
		s.handler.onPayload(payload)
	})
}

func (s *ClientHandler) OnClose(streamClose *stream.Close) {
	s.invokeFunc(func() {
		s.handler.onClose(streamClose)
	})
}

func (s *ClientHandler) OnConnectionError(conn net.Conn, err error) {
	s.invokeFunc(func() {
		s.handler.onConnectionError(conn, err)
	})
}

type clientLoggingHandler struct {
	handler *clientStateHandler
}

func (s *clientLoggingHandler) onPayload(payload *stream.Payload) {
	log.Debug("received packet", "payload", payload)
	s.handler.onPayload(payload)
}

func (s *clientLoggingHandler) onClose(streamClose *stream.Close) {
	log.Debug("received packet", "close", streamClose)
	s.handler.onClose(streamClose)
}

func (s *clientLoggingHandler) onConnectionError(conn net.Conn, err error) {
	log.Debug("on connection error", "error", err, "conn", conn)
	s.handler.onConnectionError(conn, err)
}
