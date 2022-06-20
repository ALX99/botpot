package ssh

import (
	"context"
	"net"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
)

// Session represents the database table
type Session struct {
	start   time.Time
	end     time.Time
	srcIP   string
	dstIP   string
	version string
	srcPort int
	dstPort int
}
type ipInfo struct {
	ip   string
	port int
}

// NewSession creates a new session
func NewSession(srcIP, dstIP net.Addr, version string) Session {
	s := Session{
		start:   time.Now(),
		version: version,
	}

	i := getIPInfo(srcIP)
	s.srcIP = i.ip
	s.srcPort = i.port
	i = getIPInfo(dstIP)
	s.dstIP = i.ip
	s.dstPort = i.port

	return s
}

// Insert tries to insert the data into the database
func (s *Session) Insert(tx pgx.Tx) error {
	_, err := tx.Exec(context.TODO(), `
	INSERT INTO IP(ip_address)
		VALUES ($1)
		ON CONFLICT (ip_address) DO NOTHING
`, s.srcIP)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO Session(version, src_ip, src_port, dst_ip, dst_port, start_ts, end_ts)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
`, s.version, s.srcIP, s.srcPort, s.dstIP, s.dstPort, s.start, s.end)

	return err
}

// Stop stops an active session
func (s *Session) Stop() {
	log.Info().Str("srcIP", s.srcIP).Msg("Disconnected")
	s.end = time.Now()
}

func getIPInfo(ip net.Addr) ipInfo {
	i := ipInfo{}
	switch addr := ip.(type) {
	case *net.TCPAddr:
		i.ip = addr.IP.String()
		i.port = addr.Port
	case *net.UDPAddr:
		i.ip = addr.IP.String()
		i.port = addr.Port
	}
	return i
}
