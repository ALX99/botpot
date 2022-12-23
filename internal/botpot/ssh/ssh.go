package ssh

import (
	"context"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/alx99/botpot/internal/botpot/db"
	"github.com/alx99/botpot/internal/botpot/hostprovider"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

// Server serves SSH connections from attackers
type Server struct {
	l         net.Listener
	provider  hostprovider.SSH
	cfg       *ssh.ServerConfig
	db        *db.DB
	keypaths  []string
	port      int
	lIsClosed bool
	wg        sync.WaitGroup
}

// New creates a new SSH server
func New(port int, keyPaths []string, provider hostprovider.SSH, database *db.DB) *Server {
	s := &Server{
		l:         nil,
		provider:  provider,
		cfg:       &ssh.ServerConfig{},
		db:        database,
		port:      port,
		keypaths:  keyPaths,
		lIsClosed: false,
		wg:        sync.WaitGroup{},
	}
	s.cfg = &ssh.ServerConfig{
		NoClientAuth:     true,
		MaxAuthTries:     999,
		ServerVersion:    "SSH-2.0-OpenSSH_8.9p1 Ubuntu 3",
		PasswordCallback: s.pwCallback,
	}

	return s
}

// Start starts the SSH server
func (s *Server) Start() error {
	log.Info().Msg("Starting SSH Server")
	for _, key := range s.keypaths {
		hostKey, err := readHostKey(key)
		if err != nil {
			return err
		}
		s.cfg.AddHostKey(hostKey)
	}

	var err error
	s.l, err = net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return err
	}
	log.Debug().Int("port", s.port).Msg("Started listening")

	go s.loop()
	return nil
}

// Stop stops the SSH server
func (s *Server) Stop() error {
	log.Info().Msg("Stopping SSH Server")
	s.lIsClosed = true
	err := s.l.Close()
	s.wg.Wait()
	return err
}

// nolint:ireturn //ssh.ParsePrivateKey returns interface
func readHostKey(keyPath string) (ssh.Signer, error) {
	fileBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(fileBytes)
}

func (s *Server) loop() {
	for {
		// Accept connection
		conn, err := s.l.Accept()
		if err != nil {
			if s.lIsClosed {
				break // Here we've closed the listener
			}
			log.Err(err).Msg("Could not accept connection")
			continue
		}

		// Handshake connection
		t := time.Now()
		sshConn, channelChan, reqChan, err := ssh.NewServerConn(conn, s.cfg)
		if err != nil {
			log.Err(err).Msg("Could not handshake SSH connection")
			conn.Close()
			continue
		}
		log.Debug().Str("duration", time.Since(t).String()).Msg("Connection handshaked")

		t = time.Now()
		host, ID, err := s.provider.GetHost(context.TODO())
		if err != nil {
			log.Err(err).Msg("Could not get a hold of an SSH host")
			continue
		}
		s.wg.Add(1)

		log.Debug().Str("duration", time.Since(t).String()).Msg("Host obtained")

		p := newSSHProxy(host, "root")

		// Create new client
		c := newClient(sshConn, p, channelChan)
		// Handle client
		go func() {
			defer s.wg.Done()
			c.handle(reqChan) // Blocks until client disconnects

			stdout, timing, err := s.provider.GetScriptOutput(context.TODO(), ID)
			if err != nil {
				log.Err(err).Str("id", ID).Msg("Could not get script output")
			} else {
				c.session.AddScriptOutput(stdout, timing)
			}

			err = s.provider.StopHost(context.TODO(), ID)
			if err != nil {
				log.Err(err).Str("id", ID).Msg("Could not stop host")
			}

			err = s.db.BeginTx(c.session.Insert)
			if err != nil {
				log.Err(err).Str("id", ID).Msg("Could not insert data into DB")
			}
		}()
	}
}

func (s *Server) pwCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	return nil, nil
}
