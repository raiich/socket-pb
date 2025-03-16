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
	"google.golang.org/protobuf/types/known/durationpb"
)

type ServerStreamManager struct {
	NewApp stream.NewServerHandler
}

func (m *ServerStreamManager) Setup(params *ServerSetupParams) (*ServerHandler, error) {
	writer := params.Writer
	handler := serverStateHandler{}
	handler.state.Set(&serverStreamInitialState{
		state:  &handler.state,
		writer: writer,
	})
	streamForApp := &serverStateWriter{
		state: &handler.state,
	}
	app, err := m.NewApp(streamForApp, params.Handshake)
	if err != nil {
		panic("implement me")
	}
	handler.state.Set(&serverStreamWorkingState{
		state:  &handler.state,
		writer: writer,
		conn:   params.Conn,
		app:    app,
	})
	return &ServerHandler{
		dispatcher: app.Dispatcher(),
		handler: serverLoggingHandler{
			handler: &handler,
		},
	}, nil
}

type ServerSetupParams struct {
	Conn      net.Conn
	Handshake *stream.ClientHandshake
	Writer    *packetio.ServerQueueWriter
}

type serverStateWriter struct {
	state *state.LoggingCurrentState[serverStreamState]
}

func (s *serverStateWriter) WriteServerHandshake(ctx context.Context, handshake *stream.ServerHandshake) error {
	return s.state.Get().WriteServerHandshake(ctx, handshake)
}

func (s *serverStateWriter) WritePayload(ctx context.Context, payload *stream.Payload) error {
	return s.state.Get().WritePayload(ctx, payload)
}

type serverStateHandler struct {
	state state.LoggingCurrentState[serverStreamState]
}

func (s *serverStateWriter) StartShutdown(ctx context.Context) {
	s.state.Get().StartShutdown(ctx)
}

func (s *serverStateHandler) onPayload(payload *stream.Payload) {
	s.state.Get().onPayload(payload)
}
func (s *serverStateHandler) onClose(streamClose *stream.Close) {
	s.state.Get().onClose(streamClose)
}

func (s *serverStateHandler) onConnectionError(conn net.Conn, err error) {
	s.state.Get().onConnectionError(conn, err)
}

type serverStreamState interface {
	WriteServerHandshake(ctx context.Context, handshake *stream.ServerHandshake) error
	WritePayload(ctx context.Context, payload *stream.Payload) error
	StartShutdown(ctx context.Context)
	onPayload(payload *stream.Payload)
	onClose(streamClose *stream.Close)
	onConnectionError(conn net.Conn, err error)
}

type serverStreamInitialState struct {
	state  *state.LoggingCurrentState[serverStreamState]
	writer *packetio.ServerQueueWriter
}

func (s *serverStreamInitialState) WriteServerHandshake(ctx context.Context, handshake *stream.ServerHandshake) error {
	if err := s.writer.WriteFormatVersion(ctx); err != nil {
		panic("implement me")
	}
	res := &packet.ServerHandshake{
		ProtocolVersion: 1,
		Status:          packet.HandshakeStatus_OK,
		ExtraParams:     handshake.Params,
		Settings: &packet.Settings{
			PingInterval: durationpb.New(5 * time.Second),
			PingTimeout:  durationpb.New(15 * time.Second),
		},
	}
	if err := s.writer.WriteServerHandshake(ctx, res); err != nil {
		panic("implement me")
	}
	return nil
}

func (s *serverStreamInitialState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	return s.writer.WritePacket(ctx, &packet.Packet{
		Payload: [][]byte{payload.Payload},
	})
}

func (s *serverStreamInitialState) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamInitialState) onPayload(payload *stream.Payload) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamInitialState) onClose(streamClose *stream.Close) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamInitialState) onConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}

type serverStreamWorkingState struct {
	state  *state.LoggingCurrentState[serverStreamState]
	writer *packetio.ServerQueueWriter
	conn   net.Conn
	app    stream.ServerHandler
}

func (s *serverStreamWorkingState) WriteServerHandshake(ctx context.Context, handshake *stream.ServerHandshake) error {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamWorkingState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	return s.writer.WritePacket(ctx, &packet.Packet{
		Payload: [][]byte{payload.Payload},
	})
}

func (s *serverStreamWorkingState) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamWorkingState) onPayload(payload *stream.Payload) {
	s.app.OnPayload(payload)
}

func (s *serverStreamWorkingState) onClose(streamClose *stream.Close) {
	// TODO remove stream from manager

	s.app.OnClose(&stream.Close{
		Reason: streamClose.Reason,
	})

	ctx := context.TODO()
	m := &packet.ConnectionClose{
		Reason: packet.CloseReason_NO_ERROR,
	}
	if err := s.writer.WriteConnectionClose(ctx, m); err != nil {
		panic("implement me")
	}

	closeFunc := func(writer *packetio.ServerWriter) error {
		err := s.writer.Queue.Close()
		// waiting ConnectionClose is sent to client side.
		must.NoError(s.state.AfterFunc(s.app.Dispatcher(), 500*time.Millisecond, func() {
			log.OnError(s.conn.Close())
		}))
		return err
	}
	if err := s.writer.Enqueue(ctx, closeFunc); err != nil {
		panic("implement me")
	}
	s.state.Set(&serverStreamClosingState{
		state: s.state,
	})
}

func (s *serverStreamWorkingState) onConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}

type serverStreamClosingState struct {
	state *state.LoggingCurrentState[serverStreamState]
}

func (s *serverStreamClosingState) WriteServerHandshake(ctx context.Context, handshake *stream.ServerHandshake) error {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosingState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosingState) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosingState) onPayload(payload *stream.Payload) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosingState) onClose(streamClose *stream.Close) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosingState) onConnectionError(conn net.Conn, err error) {
	s.state.Set(&serverStreamClosedState{})
}

type serverStreamClosedState struct {
}

func (s *serverStreamClosedState) WriteServerHandshake(ctx context.Context, handshake *stream.ServerHandshake) error {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosedState) WritePayload(ctx context.Context, payload *stream.Payload) error {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosedState) StartShutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosedState) onPayload(payload *stream.Payload) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosedState) onClose(streamClose *stream.Close) {
	//TODO implement me
	panic("implement me")
}

func (s *serverStreamClosedState) onConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}
