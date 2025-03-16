package packetio

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/lib/errors"
	"github.com/raiich/socket-pb/lib/task"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type ServerReader struct {
	Conn ConnReader
}

func (r *ServerReader) ReadLoop(handler ServerHandler) error {
	conn := r.Conn
	actualFormatVersion := make([]byte, len(FormatVersion1))
	if _, err := io.ReadFull(conn, actualFormatVersion); err != nil {
		return errors.Wrapf(err, "failed to read format version")
	}
	if !bytes.Equal(actualFormatVersion, FormatVersion1) {
		return errors.Newf("unsupported format version: %v", actualFormatVersion)
	}
	for {
		tag, err := binary.ReadUvarint(conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return errors.Wrapf(err, "failed to read tag")
			}
			return err
		}
		switch n, typ := protowire.DecodeTag(tag); n {
		case 2:
			if typ != protowire.BytesType {
				return errors.Newf("unsupported wire type: %v", typ)
			}
			m := &packet.ClientHandshake{}
			if err := Unmarshal(conn, m); err != nil {
				return errors.Wrapf(err, "failed to unmarshal message: %T", m)
			}
			handler.OnClientHandshake(m)
		case 3:
			if typ != protowire.BytesType {
				return errors.Newf("unsupported wire type: %v", typ)
			}
			m := &packet.Packet{}
			if err := Unmarshal(conn, m); err != nil {
				return errors.Wrapf(err, "failed to unmarshal message: %T", m)
			}
			handler.OnPacket(m)
		case 4:
			if typ != protowire.BytesType {
				return errors.Newf("unsupported wire type: %v", typ)
			}
			m := &packet.ConnectionClose{}
			if err := Unmarshal(conn, m); err != nil {
				return errors.Wrapf(err, "failed to unmarshal message: %T", m)
			}
			handler.OnConnectionClose(m)
		default:
			return errors.Newf("unexpected field: %v", n)
		}
	}
}

type ServerHandler interface {
	OnClientHandshake(msg *packet.ClientHandshake)
	OnPacket(msg *packet.Packet)
	OnConnectionClose(msg *packet.ConnectionClose)
}

type ServerLoggingHandler struct {
	Handler ServerHandler
}

func (h *ServerLoggingHandler) OnClientHandshake(msg *packet.ClientHandshake) {
	log.Debug("received packet", "packet", msg)
	h.Handler.OnClientHandshake(msg)
}

func (h *ServerLoggingHandler) OnPacket(msg *packet.Packet) {
	log.Debug("received packet", "packet", msg)
	h.Handler.OnPacket(msg)
}

func (h *ServerLoggingHandler) OnConnectionClose(msg *packet.ConnectionClose) {
	log.Debug("received packet", "packet", msg)
	h.Handler.OnConnectionClose(msg)
}

type ServerQueueWriter struct {
	Queue *task.Queue[func(*ServerWriter) error]
}

func (w *ServerQueueWriter) Enqueue(ctx context.Context, f func(*ServerWriter) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return w.Queue.Enqueue(ctx, f)
}

func (w *ServerQueueWriter) WriteFormatVersion(ctx context.Context) error {
	return w.Enqueue(ctx, func(writer *ServerWriter) error {
		return writer.WriteFormatVersion(ctx)
	})
}

func (w *ServerQueueWriter) WriteServerHandshake(ctx context.Context, m *packet.ServerHandshake) error {
	return w.Enqueue(ctx, func(writer *ServerWriter) error {
		return writer.WriteServerHandshake(ctx, m)
	})
}

func (w *ServerQueueWriter) WritePacket(ctx context.Context, m *packet.Packet) error {
	return w.Enqueue(ctx, func(writer *ServerWriter) error {
		return writer.WritePacket(ctx, m)
	})
}

func (w *ServerQueueWriter) WriteConnectionClose(ctx context.Context, m *packet.ConnectionClose) error {
	return w.Enqueue(ctx, func(writer *ServerWriter) error {
		return writer.WriteConnectionClose(ctx, m)
	})
}

type ServerWriter struct {
	Conn ConnWriter
}

func (w *ServerWriter) WriteFormatVersion(ctx context.Context) error {
	if err := w.writeFull(ctx, FormatVersion1); err != nil {
		return errors.Wrapf(err, "failed to write format version: %v", FormatVersion1)
	}
	return nil
}

func (w *ServerWriter) WriteServerHandshake(ctx context.Context, m *packet.ServerHandshake) error {
	const n = 2
	return w.write(ctx, n, m)
}

func (w *ServerWriter) WritePacket(ctx context.Context, m *packet.Packet) error {
	const n = 3
	return w.write(ctx, n, m)
}

func (w *ServerWriter) WriteConnectionClose(ctx context.Context, m *packet.ConnectionClose) error {
	const n = 4
	return w.write(ctx, n, m)
}

func (w *ServerWriter) write(ctx context.Context, n protowire.Number, m proto.Message) error {
	tag := protowire.AppendTag(nil, n, protowire.BytesType)
	if err := w.writeFull(ctx, tag); err != nil {
		return errors.Wrapf(err, "failed to write tag: %v", tag)
	}
	b, err := proto.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal message: %T", m)
	}
	lengthBytes := protowire.AppendVarint(nil, uint64(len(b)))
	if err := w.writeFull(ctx, lengthBytes); err != nil {
		return errors.Wrapf(err, "failed to write length: %v", lengthBytes)
	}
	if err := w.writeFull(ctx, b); err != nil {
		return errors.Wrapf(err, "failed to write message: %T", m)
	}
	return nil
}

func (w *ServerWriter) writeFull(_ context.Context, data []byte) error {
	if _, err := w.Conn.Write(data); err != nil {
		return errors.Wrapf(err, "failed to write data: %v", data)
	}
	return nil
}
