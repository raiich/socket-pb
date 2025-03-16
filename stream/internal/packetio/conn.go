package packetio

import (
	"encoding/binary"
	"io"

	"github.com/raiich/socket-pb/lib/errors"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var (
	formatVersionTag = protowire.AppendTag(nil, 1, protowire.VarintType)
	FormatVersion1   = protowire.AppendVarint(formatVersionTag, 1)
)

type ConnReader interface {
	io.Reader
	io.ByteReader
}

type ConnWriter interface {
	io.Writer
}

func Unmarshal(conn ConnReader, m proto.Message) error {
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
