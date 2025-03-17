package tcp

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/lib/errors"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
	"google.golang.org/protobuf/encoding/protowire"
)

var (
	errListenerClosed = errors.Newf("listener closed")
)

type Listener struct {
	base net.Listener
	conn <-chan *Conn
	ctx  context.Context
}

func (l *Listener) Accept() (*Conn, error) {
	select {
	case c, ok := <-l.conn:
		if !ok {
			return nil, errors.Newf("listener closed")
		}
		return c, nil
	case <-l.ctx.Done():
		return nil, context.Cause(l.ctx)
	}
}

func (l *Listener) Close() error {
	return l.base.Close()
}

func (l *Listener) Addr() net.Addr {
	return l.base.Addr()
}

func Listen(addr string) (*Listener, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to listen on %s", addr)
	}

	connCh := make(chan *Conn)
	ctx, cancel := context.WithCancelCause(context.Background())
	go func() {
		for {
			base, err := lis.Accept()
			if err != nil {
				cancel(err)
				return
			}
			go func() {
				conn, err := accept(ctx, base.(*net.TCPConn))
				if err != nil {
					log.OnError(base.Close())
					if !errors.Is(err, errListenerClosed) {
						log.Warn("failed to accept connection", "error", err)
					}
					return
				}
				select {
				case <-ctx.Done():
					log.OnError(base.Close())
					log.Debug("context done", "error", context.Cause(ctx))
					return
				case connCh <- conn:
					// ok
				}
			}()
		}
	}()

	return &Listener{
		base: lis,
		conn: connCh,
		ctx:  ctx,
	}, nil
}

func accept(ctx context.Context, base *net.TCPConn) (*Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	connCh := make(chan *Conn)
	errCh := make(chan error)
	go func() {
		conn, err := newServerConn(base)
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}()
	select {
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	case conn := <-connCh:
		return conn, nil
	case err := <-errCh:
		return nil, err
	}
}

func newServerConn(conn *net.TCPConn) (*Conn, error) {
	br := (*byteReader)(conn)
	actualFormatVersion := make([]byte, len(formatVersion1))
	if _, err := io.ReadFull(br, actualFormatVersion); err != nil {
		return nil, errors.Wrapf(err, "failed to read format version")
	}
	if !bytes.Equal(actualFormatVersion, formatVersion1) {
		return nil, errors.Newf("unsupported format version: %v", actualFormatVersion)
	}
	tag, err := binary.ReadUvarint(br)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, errors.Wrapf(err, "failed to read tag")
		}
		return nil, err
	}
	switch n, typ := protowire.DecodeTag(tag); n {
	case 2:
		if typ != protowire.BytesType {
			return nil, errors.Newf("unsupported wire type: %v", typ)
		}
		// ok
	default:
		return nil, errors.Newf("unexpected field: %v", n)
	}

	m := &packet.ClientHandshake{}
	if err := unmarshal(br, m); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal message: %T", m)
	}
	res := &packet.ServerHandshake{}
	if err := writeFormatVersion(conn); err != nil {
		return nil, err
	}
	if err := writeServerHandshake(conn, res); err != nil {
		return nil, err
	}
	return newConn(conn), nil
}
