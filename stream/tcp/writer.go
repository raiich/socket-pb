package tcp

import (
	"context"
	"io"
	"time"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/lib/errors"
	"github.com/raiich/socket-pb/lib/task"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type queueWriter struct {
	queue *task.Queue[func(*connWriter) error]
}

func (w *queueWriter) enqueue(ctx context.Context, f func(*connWriter) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return w.queue.Enqueue(ctx, f)
}

func (w *queueWriter) writePacket(ctx context.Context, m *packet.Packet) error {
	return w.enqueue(ctx, func(writer *connWriter) error {
		return writer.writePacket(m)
	})
}

func (w *queueWriter) writeConnectionClose(ctx context.Context, m *packet.ConnectionClose) error {
	return w.enqueue(ctx, func(writer *connWriter) error {
		return writer.writeConnectionClose(m)
	})
}

func (w *queueWriter) shutdown(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	closeFunc := func(_ *connWriter) error {
		return w.queue.Close()
	}
	if err := w.queue.Enqueue(ctx, closeFunc); err != nil {
		log.Error("failed to enqueue close function", "error", err)
		log.OnError(w.queue.Close())
	}
}

type connWriter struct {
	conn io.Writer
}

func (w *connWriter) writePacket(m *packet.Packet) error {
	const n = 3
	return w.write(n, m)
}

func (w *connWriter) writeConnectionClose(m *packet.ConnectionClose) error {
	const n = 4
	return w.write(n, m)
}

func (w *connWriter) write(n protowire.Number, m proto.Message) error {
	return write(w.conn, n, m)
}

func writeFormatVersion(w io.Writer) error {
	if _, err := w.Write(formatVersion1); err != nil {
		return errors.Wrapf(err, "failed to write format version: %v", formatVersion1)
	}
	return nil
}

func writeClientHandshake(w io.Writer, m *packet.ClientHandshake) error {
	const n = 2
	return write(w, n, m)
}

func writeServerHandshake(w io.Writer, m *packet.ServerHandshake) error {
	const n = 2
	return write(w, n, m)
}

func write(w io.Writer, n protowire.Number, m proto.Message) error {
	tag := protowire.AppendTag(nil, n, protowire.BytesType)
	if _, err := w.Write(tag); err != nil {
		return errors.Wrapf(err, "failed to write tag: %v", tag)
	}
	b, err := proto.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal message: %T", m)
	}
	lengthBytes := protowire.AppendVarint(nil, uint64(len(b)))
	if _, err := w.Write(lengthBytes); err != nil {
		return errors.Wrapf(err, "failed to write length: %v", lengthBytes)
	}
	if _, err := w.Write(b); err != nil {
		return errors.Wrapf(err, "failed to write message: %T", m)
	}
	return nil
}
