package stream

import (
	"io"
	"net"

	"github.com/raiich/socket-pb/lib/state"
)

type Stream struct {
	state state.LoggingCurrentState[streamState]
}

func (s *Stream) Read(p []byte) (n int, err error) {
	return s.state.Get().read(p)
}

func (s *Stream) Write(p []byte) (n int, err error) {
	return s.state.Get().write(p)
}

func New(conn Conn) *Stream {
	ret := &Stream{}
	ret.state.Set(&streamStateWorking{
		state: &ret.state,
		conn:  conn,
	})
	return ret
}

type Conn interface {
	Write(p []byte) (n int, err error)
	Read(p []byte) (n int, err error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type streamState interface {
	read(p []byte) (n int, err error)
	write(p []byte) (n int, err error)
}

type streamStateWorking struct {
	state *state.LoggingCurrentState[streamState]
	conn  Conn
}

func (s *streamStateWorking) read(p []byte) (n int, err error) {
	n, err = s.conn.Read(p)
	if err != nil {
		s.state.Set(&streamStateClosed{})
	}
	return
}

func (s *streamStateWorking) write(p []byte) (n int, err error) {
	n, err = s.conn.Write(p)
	if err != nil {
		s.state.Set(&streamStateClosed{})
	}
	return
}

type streamStateClosed struct{}

func (s *streamStateClosed) read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (s *streamStateClosed) write(p []byte) (n int, err error) {
	return 0, io.EOF
}
