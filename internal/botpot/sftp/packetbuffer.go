package sftp

import (
	"encoding/binary"
	"errors"
)

var (
	errShortPacket = errors.New("short packet")
)

type packetBuffer struct {
	buf []byte
	pos uint64
	len uint64
}

func newPacketBuffer(data []byte) packetBuffer {
	return packetBuffer{
		buf: data,
		pos: 0,
		len: uint64(len(data)),
	}
}

func (pb *packetBuffer) getRemainingBytes() []byte {
	return pb.buf[pb.pos:]
}

func (pb *packetBuffer) readUint32() (uint32, error) {
	pb.pos += 4
	if pb.pos > pb.len {
		return 0, errShortPacket
	}
	return binary.BigEndian.Uint32(pb.buf[pb.pos-4 : pb.pos]), nil
}

func (pb *packetBuffer) readUint64() (uint64, error) {
	pb.pos += 8
	if pb.pos > pb.len {
		return 0, errShortPacket
	}
	return binary.BigEndian.Uint64(pb.buf[pb.pos-8 : pb.pos]), nil
}

func (pb *packetBuffer) readUTF8() (string, error) {
	if pb.pos+4 > pb.len {
		return "", errShortPacket
	}
	pb.pos += 4

	strLen := uint64(binary.BigEndian.Uint32(pb.buf[pb.pos-4 : pb.pos]))
	if pb.pos+strLen > pb.len {
		return "", errShortPacket
	}
	pb.pos += strLen

	return string(pb.buf[pb.pos-strLen : pb.pos]), nil
}

