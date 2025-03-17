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

func Dial(addr string, opts ...DialOption) (*Conn, error) {
	base, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	settings := newClientSettings(opts...)

	ch := make(chan *Conn)
	errCh := make(chan error)
	go func() {
		conn, err := dial(base.(*net.TCPConn), settings)
		if err != nil {
			errCh <- err
			return
		}
		ch <- conn
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		log.OnError(base.Close())
		return nil, errors.Newf("dial timeout")
	case conn := <-ch:
		return conn, nil
	case err := <-errCh:
		log.OnError(base.Close())
		return nil, err
	}
}

func dial(conn *net.TCPConn, settings *clientSettings) (*Conn, error) {
	handshake := &packet.ClientHandshake{
		ProtocolVersion: 1,
		ExtraParams:     settings.ExtraParams,
	}
	if err := writeFormatVersion(conn); err != nil {
		return nil, err
	}
	if err := writeClientHandshake(conn, handshake); err != nil {
		return nil, err
	}

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

	m := &packet.ServerHandshake{}
	if err := unmarshal(br, m); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal message: %T", m)
	}
	switch m.Status {
	case packet.HandshakeStatus_OK:
		// ok
	case packet.HandshakeStatus_RETRY_LATER:
		return nil, errors.Newf("server is busy")
	case packet.HandshakeStatus_UNACCEPTED:
		return nil, errors.Newf("unaccepted connection")
	}
	return newConn(conn), nil
}

type DialOption interface {
	apply(settings *clientSettings)
}

type clientOptionFunc func(settings *clientSettings)

func (f clientOptionFunc) apply(settings *clientSettings) {
	f(settings)
}

type clientSettings struct {
	ExtraParams map[string][]byte
}

func newClientSettings(opts ...DialOption) *clientSettings {
	settings := &clientSettings{}
	for _, opt := range opts {
		opt.apply(settings)
	}
	return settings
}

func WithExtraParams(extraParams map[string][]byte) DialOption {
	return clientOptionFunc(func(settings *clientSettings) {
		settings.ExtraParams = extraParams
	})
}
