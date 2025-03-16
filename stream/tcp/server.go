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

type Server struct {
	streams internal.ServerStreamManager
}

func (s *Server) Serve(lis net.Listener) error {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return errors.Wrapf(err, "failed to accept")
		}
		s.serve(conn)
	}
}

func (s *Server) serve(conn net.Conn) {
	go func() {
		// TODO recover
		defer func() {
			// ensure conn is closed (writer side may be alive)
			log.Debug("closing reader conn")
			log.OnError(conn.Close())
		}()

		stateHandler := serverStateHandler{}
		stateHandler.state.Set(&serverConnInitialState{
			state:   &stateHandler.state,
			conn:    conn,
			streams: &s.streams,
		})
		handler := &packetio.ServerLoggingHandler{
			Handler: &stateHandler,
		}
		r := &packetio.ServerReader{
			Conn: bufio.NewReader(conn),
		}
		if err := r.ReadLoop(handler); err != nil {
			stateHandler.OnConnectionError(conn, err)
		}
	}()
}

func NewServer(newApp stream.NewServerHandler) *Server {
	sv := &Server{
		streams: internal.ServerStreamManager{
			NewApp: newApp,
		},
	}
	return sv
}

type serverStateHandler struct {
	state state.LoggingCurrentState[serverConnState]
}

func (s *serverStateHandler) OnClientHandshake(msg *packet.ClientHandshake) {
	s.state.Get().OnClientHandshake(msg)
}

func (s *serverStateHandler) OnPacket(msg *packet.Packet) {
	s.state.Get().OnPacket(msg)
}

func (s *serverStateHandler) OnConnectionClose(msg *packet.ConnectionClose) {
	s.state.Get().OnConnectionClose(msg)
}

func (s *serverStateHandler) OnConnectionError(conn net.Conn, err error) {
	s.state.Get().OnConnectionError(conn, err)
}

type serverConnState interface {
	OnClientHandshake(msg *packet.ClientHandshake)
	OnPacket(msg *packet.Packet)
	OnConnectionClose(msg *packet.ConnectionClose)
	OnConnectionError(conn net.Conn, err error)
}

type serverConnInitialState struct {
	state   *state.LoggingCurrentState[serverConnState]
	conn    net.Conn
	streams *internal.ServerStreamManager
}

func (s *serverConnInitialState) OnClientHandshake(msg *packet.ClientHandshake) {
	conn := s.conn
	queue := task.NewQueue[func(writer *packetio.ServerWriter) error](0)
	go func() {
		// TODO recover
		bw := bufio.NewWriter(conn)
		w := &packetio.ServerWriter{
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

	writer := &packetio.ServerQueueWriter{
		Queue: queue,
	}
	params := internal.ServerSetupParams{
		Conn: conn,
		Handshake: &stream.ClientHandshake{
			Params: msg.ExtraParams,
		},
		Writer: writer,
	}
	handler, err := s.streams.Setup(&params)
	if err != nil {
		panic("implement me")
	}
	s.state.Set(&serverConnEstablishedState{
		state:   s.state,
		conn:    conn,
		handler: handler,
	})
}

func (s *serverConnInitialState) OnPacket(msg *packet.Packet) {
	//TODO implement me
	panic("implement me")
}

func (s *serverConnInitialState) OnConnectionClose(msg *packet.ConnectionClose) {
	//TODO implement me
	panic("implement me")
}

func (s *serverConnInitialState) OnConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}

type serverConnEstablishedState struct {
	state   *state.LoggingCurrentState[serverConnState]
	conn    net.Conn
	handler *internal.ServerHandler
}

func (s *serverConnEstablishedState) OnClientHandshake(msg *packet.ClientHandshake) {
	//TODO implement me
	panic("implement me")
}

func (s *serverConnEstablishedState) OnPacket(msg *packet.Packet) {
	for _, payload := range msg.Payload {
		s.handler.OnPayload(&stream.Payload{
			Payload: payload,
		})
	}
}

func (s *serverConnEstablishedState) OnConnectionClose(msg *packet.ConnectionClose) {
	s.handler.OnClose(&stream.Close{
		Reason: msg.Reason,
	})
}

func (s *serverConnEstablishedState) OnConnectionError(conn net.Conn, err error) {
	if s.conn != conn {
		panic(fmt.Sprintf("unexpected connection: %v != %v", s.conn, conn))
	}
	log.Warn("connection error", "conn", conn, "error", err, "state", s.state.Get())
	s.state.Set(&serverConnClosedState{})
	s.handler.OnConnectionError(conn, err)
}

type serverConnClosedState struct {
}

func (s *serverConnClosedState) OnClientHandshake(msg *packet.ClientHandshake) {
	//TODO implement me
	panic("implement me")
}

func (s *serverConnClosedState) OnPacket(msg *packet.Packet) {
	//TODO implement me
	panic("implement me")
}

func (s *serverConnClosedState) OnConnectionClose(msg *packet.ConnectionClose) {
	//TODO implement me
	panic("implement me")
}

func (s *serverConnClosedState) OnConnectionError(conn net.Conn, err error) {
	//TODO implement me
	panic("implement me")
}
