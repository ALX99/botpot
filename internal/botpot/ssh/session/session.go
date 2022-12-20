package session

import (
	"context"
	"net"
	"time"

	"github.com/alx99/botpot/internal/botpot/ssh/channel"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// Session represents the database table
type Session struct {
	start   time.Time
	end     time.Time
	srcIP   string
	dstIP   string
	version string
	stdout  string
	timing  string
	l       zerolog.Logger
	channels     []*channel.Channel
	srcPort int
	dstPort int
}
type ipInfo struct {
	ip   string
	port int
}

// NewSession creates a new session
func NewSession(srcIP, dstIP net.Addr, version string, l zerolog.Logger) Session {
	s := Session{
		start:   time.Now(),
		version: version,
		l:       l,
		channels:     []*channel.Channel{},
	}

	i := getIPInfo(srcIP)
	s.srcIP = i.ip
	s.srcPort = i.port
	i = getIPInfo(dstIP)
	s.dstIP = i.ip
	s.dstPort = i.port

	return s
}

// AddScriptOutput adds the script output to the session
func (s *Session) AddScriptOutput(stdout, timing string) {
	s.stdout = stdout
	s.timing = timing
}

// AddChannel adds a channel to the session
func (s *Session) AddChannel(ch *channel.Channel) {
	s.channels = append(s.channels, ch)
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
	INSERT INTO Session(version, src_ip, src_port, dst_ip, dst_port, start_ts, end_ts, stdout, timing)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, s.version, s.srcIP, s.srcPort, s.dstIP, s.dstPort, s.start, s.end, s.stdout, s.timing)
	if err != nil {
		return err
	}

	for _, ch := range s.channels {
		err = ch.Insert(tx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop stops an active session
func (s *Session) Stop() {
	s.l.Info().Msg("Disconnected")
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
