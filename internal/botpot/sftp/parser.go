package sftp

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/rs/zerolog"
)

// Parser can parse SFTP client messages
type Parser struct {
	data *bytes.Buffer
	l    zerolog.Logger
}

// NewParser creates a new SFTP server.
func NewParser(buf []byte, logger zerolog.Logger) Parser {
	return Parser{data: bytes.NewBuffer(buf), l: logger}
}

// Parse parses the data provided
func (s *Parser) Parse() error {
	packets, err := s.readAllPackets()
	if err != nil {
		return err
	}

	s.l.Debug().Int("count", len(packets)).Msg("Succesfully parsed all SFTP packets")
	return nil
}

func (s *Parser) readAllPackets() ([]sftpPacket, error) {
	var packets []sftpPacket
	for s.data.Len() > 0 {
		p, err := readPacket(s.data)
		if err != nil {
			return nil, err
		}
		packets = append(packets, p)
	}

	return packets, nil
}

// sftpPacket structures an SFTP packet
// https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02#section-3
type sftpPacket struct {
	data   []byte
	length uint32
	pType  byte
}

// readPacket reads a single SFTP packet
func readPacket(r io.Reader) (sftpPacket, error) {
	var len uint32
	err := binary.Read(r, binary.BigEndian, &len)
	if err != nil {
		return sftpPacket{}, err
	}

	var pType byte
	err = binary.Read(r, binary.BigEndian, &pType)
	if err != nil {
		return sftpPacket{}, err
	}

	buf := make([]byte, len-1) // -1 since length includes type
	err = binary.Read(r, binary.BigEndian, buf)
	if err != nil {
		return sftpPacket{}, err
	}

	return sftpPacket{length: len, pType: pType, data: buf}, nil
}
