package sftp

import (
	"bytes"
	"encoding/binary"
)

type packetBuffer struct {
	buf *bytes.Buffer
	len int
}

func newPacketBuffer(data []byte) packetBuffer {
	return packetBuffer{
		buf: bytes.NewBuffer(data),
		len: len(data),
	}
}

func (pb *packetBuffer) getRemainingBytes() []byte {
	v := make([]byte, pb.len)
	if err := binary.Read(pb.buf, binary.BigEndian, &v); err != nil {
		return []byte{}
	}
	return v
}

func (pb *packetBuffer) readUint8() (uint8, error) {
	var v uint8
	if err := binary.Read(pb.buf, binary.BigEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

func (pb *packetBuffer) readUint32() (uint32, error) {
	var v uint32
	if err := binary.Read(pb.buf, binary.BigEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

func (pb *packetBuffer) readUint64() (uint64, error) {
	var v uint64
	if err := binary.Read(pb.buf, binary.BigEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

func (pb *packetBuffer) readInt64() (int64, error) {
	var v int64
	if err := binary.Read(pb.buf, binary.BigEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

func (pb *packetBuffer) readUTF8() (string, error) {
	var strLen uint32
	if err := binary.Read(pb.buf, binary.BigEndian, &strLen); err != nil {
		return "", err
	}

	v := make([]byte, strLen)
	if err := binary.Read(pb.buf, binary.BigEndian, &v); err != nil {
		return "", err
	}
	return string(v), nil
}
