package tcp

import (
	"bufio"
	"fmt"
	"net"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/lib/errors"
	"github.com/raiich/socket-pb/lib/state"
	"github.com/raiich/socket-pb/lib/task"
	"github.com/raiich/socket-pb/stream"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
	"github.com/raiich/socket-pb/stream/internal"
	"github.com/raiich/socket-pb/stream/internal/packetio"
)

func Dial(dispatcher stream.Dispatcher, addr string, newApp stream.NewClientHandler, opts ...ClientOption) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c := client{
		dispatcher: dispatcher,
		newApp:     newApp,
		settings:   newClientSettings(opts...),
	}
	go c.launch(conn)
	return nil
}

type client struct {
	dispatcher stream.Dispatcher
	newApp     stream.NewClientHandler
	settings   *clientSettings
}

func (c *client) launch(conn net.Conn) {
	// TODO recover
	defer func() {
		// ensure conn is closed (writer side may be alive)
		log.Debug("closing reader conn")
		log.OnError(conn.Close())
	}()

	queue := task.NewQueue[func(writer *packetio.ClientWriter) error](0)
	writer := &packetio.ClientQueueWriter{
		Queue: queue,
	}
	stateHandler := clientStateHandler{}
	stateHandler.state.Set(&clientConnInitialState{
		state:  &stateHandler.state,
		conn:   conn,
		client: c,
		writer: writer,
	})

	go func() {
		// TODO recover
		bw := bufio.NewWriter(conn)
		w := &packetio.ClientWriter{
			Conn: bw,
		}
		for {
			doWrite, err := queue.Dequeue()
			if err != nil {
				if errors.Is(err, task.ErrClosedQueue) {
					return
				}
				log.Warn("failed to dequeue", "error", err)
				return
			}
			if err := doWrite(w); err != nil {
				log.Warn("failed to write", "error", err)
				// ensure conn is closed (reader side may be alive)
				_ = conn.Close()
				return
			}
			if err := bw.Flush(); err != nil {
				log.Warn("failed to flush", "error", err)
				// ensure conn is closed (reader side may be alive)
				_ = conn.Close()
				return
			}
		}
	}()

	ctx := c.dispatcher.Context()
	if err := writer.WriteFormatVersion(ctx); err != nil {
		log.Warn("failed to write format version", "error", err)
		log.OnError(conn.Close())
	}
	handshake := &packet.ClientHandshake{
		ProtocolVersion: 1,
		ExtraParams:     c.settings.ExtraParams,
	}
	if err := writer.WriteClientHandshake(ctx, handshake); err != nil {
		log.Warn("failed to write handshake", "error", err)
		log.OnError(conn.Close())
		return
	}

	r := &packetio.ClientReader{
		Conn: bufio.NewReader(conn),
	}
	handler := &packetio.ClientLoggingHandler{
		Handler: &stateHandler,
	}
	if err := r.ReadLoop(handler); err != nil {
		stateHandler.OnConnectionError(conn, err)
	}
}

type ClientOption interface {
	apply(settings *clientSettings)
}

type clientOptionFunc func(settings *clientSettings)

func (f clientOptionFunc) apply(settings *clientSettings) {
	f(settings)
}

type clientSettings struct {
	ExtraParams map[string][]byte
}

func newClientSettings(opts ...ClientOption) *clientSettings {
	settings := &clientSettings{}
	for _, opt := range opts {
		opt.apply(settings)
	}
	return settings
}

func WithExtraParams(extraParams map[string][]byte) ClientOption {
	return clientOptionFunc(func(settings *clientSettings) {
		settings.ExtraParams = extraParams
	})
}

type clientStateHandler struct {
	state state.LoggingCurrentState[clientConnState]
}

func (s *clientStateHandler) OnServerHandshake(msg *packet.ServerHandshake) {
	s.state.Get().OnServerHandshake(msg)
}

func (s *clientStateHandler) OnPacket(msg *packet.Packet) {
	s.state.Get().OnPacket(msg)
}

func (s *clientStateHandler) OnConnectionClose(msg *packet.ConnectionClose) {
	s.state.Get().OnConnectionClose(msg)
}

func (s *clientStateHandler) OnConnectionError(conn net.Conn, err error) {
	s.state.Get().OnConnectionError(conn, err)
}

type clientConnState interface {
	OnServerHandshake(msg *packet.ServerHandshake)
	OnPacket(msg *packet.Packet)
	OnConnectionClose(msg *packet.ConnectionClose)
	OnConnectionError(conn net.Conn, err error)
}

type clientConnInitialState struct {
	state  *state.LoggingCurrentState[clientConnState]
	conn   net.Conn
	client *client
	writer *packetio.ClientQueueWriter
}

func (s *clientConnInitialState) OnServerHandshake(msg *packet.ServerHandshake) {
	params := internal.ClientSetupParams{
		Conn:   s.conn,
		Writer: s.writer,
		Handshake: &stream.ServerHandshake{
			Status: msg.Status,
			Params: msg.ExtraParams,
		},
	}
	handler, err := internal.SetupClient(s.client.dispatcher, s.client.newApp, &params)
	if err != nil {
		panic("implement me")
	}
	s.state.Set(&clientConnEstablishedState{
		state:   s.state,
		conn:    s.conn,
		handler: handler,
	})
}

func (s *clientConnInitialState) OnPacket(msg *packet.Packet) {
	//TODO implement me
	panic("implement me")
}

func (s *clientConnInitialState) OnConnectionClose(msg *packet.ConnectionClose) {
	//TODO implement me
	panic("implement me")
}

func (s *clientConnInitialState) OnConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}

type clientConnEstablishedState struct {
	state   *state.LoggingCurrentState[clientConnState]
	conn    net.Conn
	handler *internal.ClientHandler
}

func (s *clientConnEstablishedState) OnServerHandshake(msg *packet.ServerHandshake) {
	//TODO implement me
	panic("implement me")
}

func (s *clientConnEstablishedState) OnPacket(msg *packet.Packet) {
	for _, payload := range msg.Payload {
		s.handler.OnPayload(&stream.Payload{
			Payload: payload,
		})
	}
}

func (s *clientConnEstablishedState) OnConnectionClose(msg *packet.ConnectionClose) {
	s.handler.OnClose(&stream.Close{
		Reason: msg.Reason,
	})
}

func (s *clientConnEstablishedState) OnConnectionError(conn net.Conn, err error) {
	if s.conn != conn {
		panic(fmt.Sprintf("unexpected connection: %v != %v", s.conn, conn))
	}

	errorState := s.state.Get()
	log.Warn("connection error", "conn", conn, "error", err, "state", errorState)
	s.handler.OnConnectionError(conn, err)

	s.state.Set(&clientConnClosedState{})
}

type clientConnClosedState struct {
}

func (s *clientConnClosedState) OnServerHandshake(msg *packet.ServerHandshake) {
	//TODO implement me
	panic("implement me")
}

func (s *clientConnClosedState) OnPacket(msg *packet.Packet) {
	//TODO implement me
	panic("implement me")
}

func (s *clientConnClosedState) OnConnectionClose(msg *packet.ConnectionClose) {
	//TODO implement me
	panic("implement me")
}

func (s *clientConnClosedState) OnConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}
