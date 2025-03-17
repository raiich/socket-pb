package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"

	"github.com/raiich/socket-pb/internal/log"
	"github.com/raiich/socket-pb/lib/errors"
	"github.com/raiich/socket-pb/lib/task"
	packet "github.com/raiich/socket-pb/stream/generated/go/packet/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var (
	formatVersionTag = protowire.AppendTag(nil, 1, protowire.VarintType)
	formatVersion1   = protowire.AppendVarint(formatVersionTag, 1)
)

type Conn struct {
	reader *bufio.Reader
	prev   []byte
	writer *queueWriter
	local  net.Addr
	remote net.Addr
}

func (c *Conn) Read(b []byte) (n int, err error) {
	if len(c.prev) > 0 {
		n = copy(b, c.prev)
		c.prev = c.prev[n:]
		return n, nil
	}

	tag, err := binary.ReadUvarint(c.reader)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return 0, errors.Wrapf(err, "failed to read tag")
		}
		return 0, err
	}
	switch num, typ := protowire.DecodeTag(tag); num {
	case 3:
		if typ != protowire.BytesType {
			return 0, errors.Newf("unsupported wire type: %v", typ)
		}
		m := &packet.Packet{}
		if err := unmarshal(c.reader, m); err != nil {
			return 0, errors.Wrapf(err, "failed to unmarshal message: %T", m)
		}
		n = copy(b, m.Payload)
		if n < len(m.Payload) {
			c.prev = m.Payload[n:]
		}
		return n, nil
	case 4:
		if typ != protowire.BytesType {
			return 0, errors.Newf("unsupported wire type: %v", typ)
		}
		m := &packet.ConnectionClose{}
		if err := unmarshal(c.reader, m); err != nil {
			return 0, errors.Wrapf(err, "failed to unmarshal message: %T", m)
		}
		return 0, io.EOF
	default:
		return 0, errors.Newf("unexpected field: %v", num)
	}
}

func (c *Conn) Write(b []byte) (n int, err error) {
	ctx := context.TODO()
	p := &packet.Packet{
		Payload: b,
	}
	if err := c.writer.writePacket(ctx, p); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *Conn) Close() error {
	err := c.writeConnectionClose()
	ctx := context.TODO()
	c.writer.shutdown(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to write connection close")
	}
	return nil
}

func (c *Conn) writeConnectionClose() error {
	ctx := context.TODO()
	p := &packet.ConnectionClose{
		Reason: packet.CloseReason_NO_ERROR,
	}
	return c.writer.writeConnectionClose(ctx, p)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.local
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remote
}

func newConn(conn *net.TCPConn) *Conn {
	bw := bufio.NewWriter(conn)
	w := &connWriter{
		conn: bw,
	}
	queue := task.NewQueue[func(writer *connWriter) error](0)
	go func() {
		// TODO recover
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

	return &Conn{
		reader: bufio.NewReader(conn),
		writer: &queueWriter{
			queue: queue,
		},
		local:  conn.LocalAddr(),
		remote: conn.RemoteAddr(),
	}
}

func unmarshal(conn connReader, m proto.Message) error {
	l, err := binary.ReadUvarint(conn)
	if err != nil {
		return errors.Wrapf(err, "failed to read length")
	}
	buf := make([]byte, l)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.Wrapf(err, "failed to read bytes from stream")
	}
	if err := proto.Unmarshal(buf, m); err != nil {
		return errors.Wrapf(err, "failed to unmarshal message: %T", m)
	}
	return nil
}

type connReader interface {
	io.Reader
	io.ByteReader
}

type byteReader net.TCPConn

func (r *byteReader) ReadByte() (byte, error) {
	buf := [1]byte{}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return buf[0], nil
}
