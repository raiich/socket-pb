package internal

import (
	"context"
	"net"
	"time"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/lib/must"
	"github.com/raiich/socket-pb/lib/state"
	"github.com/raiich/socket-pb/stream"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
	"github.com/raiich/socket-pb/stream/internal/packetio"
)

func SetupClient(dispatcher stream.Dispatcher, newApp stream.NewClientHandler, params *ClientSetupParams) (*ClientHandler, error) {
	writer := params.Writer
	handler := clientStateHandler{}
	handler.state.Set(&clientStreamInitialState{
		writer: writer,
	})
	streamForApp := &clientStateWriter{
		state: &handler.state,
	}
	app, err := newApp(streamForApp, params.Handshake)
	if err != nil {
		panic("implement me")
	}
	handler.state.Set(&clientStreamWorkingState{
		state:      &handler.state,
		dispatcher: dispatcher,
		writer:     writer,
		app:        app,
		conn:       params.Conn,
	})
	return &ClientHandler{
		dispatcher: dispatcher,
		handler: clientLoggingHandler{
			handler: &handler,
		},
	}, nil
}

type ClientSetupParams struct {
	Conn      net.Conn
	Writer    *packetio.ClientQueueWriter
	Handshake *stream.ServerHandshake
}

type clientStateWriter struct {
	state *state.LoggingCurrentState[clientStreamState]
	//dispatcher stream.Dispatcher
	//conn       net.Conn
	//writer     *packetio.ClientQueueWriter
}

func (s *clientStateWriter) WritePayload(ctx context.Context, payload *stream.Payload) error {
	return s.state.Get().WritePayload(ctx, payload)
}

func (s *clientStateWriter) StartShutdown(ctx context.Context) {
	s.state.Get().StartShutdown(ctx)
}

type clientStateHandler struct {
	state state.LoggingCurrentState[clientStreamState]
}

func (s *clientStateHandler) onPayload(payload *stream.Payload) {
	s.state.Get().onPayload(payload)
}

func (s *clientStateHandler) onClose(streamClose *stream.Close) {
	s.state.Get().onClose(streamClose)
}

func (s *clientStateHandler) onConnectionError(conn net.Conn, err error) {
	s.state.Get().onConnectionError(conn, err)
}

type clientStreamState interface {
	WritePayload(ctx context.Context, payload *stream.Payload) error
	StartShutdown(ctx context.Context)
	onPayload(payload *stream.Payload)
	onClose(streamClose *stream.Close)
	onConnectionError(conn net.Conn, err error)
}

type clientStreamInitialState struct {
	writer *packetio.ClientQueueWriter
}

func (s *clientStreamInitialState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	return s.writer.WritePacket(ctx, &packet.Packet{
		Payload: [][]byte{payload.Payload},
	})
}

func (s *clientStreamInitialState) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamInitialState) onPayload(payload *stream.Payload) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamInitialState) onClose(streamClose *stream.Close) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamInitialState) onConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}

type clientStreamWorkingState struct {
	state      *state.LoggingCurrentState[clientStreamState]
	dispatcher stream.Dispatcher
	writer     *packetio.ClientQueueWriter
	app        stream.ClientHandler
	conn       net.Conn
}

func (s *clientStreamWorkingState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	return s.writer.WritePacket(ctx, &packet.Packet{
		Payload: [][]byte{payload.Payload},
	})
}

func (s *clientStreamWorkingState) StartShutdown(ctx context.Context) {
	res := &packet.ConnectionClose{
		Reason: packet.CloseReason_NO_ERROR,
	}
	if err := s.writer.WriteConnectionClose(ctx, res); err != nil {
		log.Warn("failed to write ConnectionClose", "error", err)
		panic("implement me")
	}

	closeFunc := func(writer *packetio.ClientWriter) error {
		err := s.writer.Queue.Close()
		s.state.Set(&clientStreamClosingState{
			state: s.state,
			conn:  s.conn,
			app:   s.app,
		})
		// waiting ConnectionClose is sent to client side.
		must.NoError(s.state.AfterFunc(s.dispatcher, 500*time.Millisecond, func() {
			panic("implement me")
		}))
		return err
	}
	if err := s.writer.Enqueue(ctx, closeFunc); err != nil {
		panic("implement me")
	}
}

func (s *clientStreamWorkingState) onPayload(payload *stream.Payload) {
	s.app.OnPayload(payload)
}

func (s *clientStreamWorkingState) onClose(streamClose *stream.Close) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamWorkingState) onConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}

type clientStreamClosingState struct {
	state *state.LoggingCurrentState[clientStreamState]
	conn  net.Conn
	app   stream.ClientHandler
}

func (s *clientStreamClosingState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamClosingState) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamClosingState) onPayload(payload *stream.Payload) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamClosingState) onClose(streamClose *stream.Close) {
	s.state.Set(&clientStreamClosedState{})
	log.OnError(s.conn.Close())
	s.app.OnClose(streamClose)
	return
}

func (s *clientStreamClosingState) onConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}

type clientStreamClosedState struct {
}

func (s *clientStreamClosedState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamClosedState) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamClosedState) onPayload(payload *stream.Payload) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamClosedState) onClose(streamClose *stream.Close) {
	//TODO implement me
	panic("implement me")
}

func (s *clientStreamClosedState) onConnectionError(conn net.Conn, err error) {
	log.Debug("connection closed successfully", "conn", conn, "error", err)
}
