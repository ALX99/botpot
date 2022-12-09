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
	err = s.parseInfo(packets)
	if err != nil {
		return err
	}

	s.l.Debug().Int("count", len(packets)).Msg("Succesfully parsed all SFTP packets")
	return nil
}

func (s *Parser) readAllPackets() ([]packet, error) {
	var packets []packet
	for s.data.Len() > 0 {
		p, err := readPacket(s.data)
		if err != nil {
			return nil, err
		}
		packets = append(packets, p)
	}

	return packets, nil
}

func (s *Parser) parseInfo(packets []packet) error {
	for _, packet := range packets {
		switch packet.pType {
		case sshFXPInit:
			p := Init{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("SFTP server version %d", p.Version)
		case sshFXPOpen:
			p := Open{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("Open: %s", p.Filename)
		case sshFXPClose:
			p := Close{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("Close: %d %s", p.RequestID, p.Handle)
		case sshFXPRead:
			p := Read{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("Read: %d %d", p.RequestID, p.Offset)
		case sshFXPWrite:
			p := Write{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("Write: %d %s %d", p.RequestID, p.Handle, p.Offset)
		case sshFXPMkdir:
			p := Mkdir{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("Mkdir: %d %s", p.RequestID, p.Path)
		case sshFXPOpenDir:
			p := OpenDir{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("Opendir: %d %s", p.RequestID, p.Path)
		case sshFXPReadDir:
			p := ReadDir{}
			if err := p.UnmarshalBinary(packet.data); err != nil {
				return err
			}
			s.l.Info().Msgf("Readdir: %d %s", p.RequestID, p.Handle)
		default:
			// s.l.Warn().Uint8("type", p.pType).Msg("Did not understand SFTP packet")
		}
	}
	return nil
}

// readPacket reads a single SFTP packet
func readPacket(r io.Reader) (packet, error) {
	var len uint32
	err := binary.Read(r, binary.BigEndian, &len)
	if err != nil {
		return packet{}, err
	}

	var pType byte
	err = binary.Read(r, binary.BigEndian, &pType)
	if err != nil {
		return packet{}, err
	}

	buf := make([]byte, len-1) // -1 since length includes type
	err = binary.Read(r, binary.BigEndian, buf)
	if err != nil {
		return packet{}, err
	}

	return packet{length: len, pType: pType, data: buf}, nil
}
