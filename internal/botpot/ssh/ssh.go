package ssh

import (
	"io/ioutil"
	"net"
	"strconv"

	"github.com/alx99/botpot/internal/botpot/db"
	"github.com/alx99/botpot/internal/hostprovider"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	l         net.Listener
	provider  hostprovider.SSH
	cfg       *ssh.ServerConfig
	db        *db.DB
	port      int
	lIsClosed bool
}

// New creates a new SSH server
func New(port int, provider hostprovider.SSH, database *db.DB) *Server {
	s := &Server{
		port:     port,
		provider: provider,
		db:       database,
	}
	s.cfg = &ssh.ServerConfig{
		NoClientAuth:     true,
		MaxAuthTries:     999,
		ServerVersion:    "SSH-2.0-OpenSSH_8.8",
		PasswordCallback: s.pwCallback,
	}

	return s
}

// Start starts the SSH server
func (s *Server) Start() error {
	log.Info().Msg("Starting SSH Server")
	hostKey, err := readHostKey("./key")
	if err != nil {
		return err
	}
	s.cfg.AddHostKey(hostKey)

	s.l, err = net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return err
	}
	log.Debug().Msgf("Started listening on :%d", s.port)

	go s.loop()
	return nil
}

// Stop stops the SSH server
func (s *Server) Stop() error {
	log.Info().Msg("Stopping SSH Server")
	s.lIsClosed = true
	return s.l.Close()
}

func readHostKey(keyPath string) (ssh.Signer, error) {
	fileBytes, err := ioutil.ReadFile(keyPath)
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
		sshConn, channelChan, reqChan, err := ssh.NewServerConn(conn, s.cfg)
		if err != nil {
			log.Err(err).Msg("Could not handshake SSH connection")
			conn.Close()
			continue
		}

		host, ID, err := s.provider.GetHost()
		if err != nil {
			log.Err(err).Msg("Could not get a hold of an SSH host")
			continue
		}
		p := newSSHProxy(host, "panda", "password")

		// Create new client
		c := newClient(sshConn, p, s.db, channelChan)
		// Handle client
		go func() {
			// Blocks until client disconnects
			c.handle(reqChan)

			err := s.provider.StopHost(ID)
			if err != nil {
				log.Err(err).Msgf("Could not stop host %s", ID)
			}

			err = s.db.BeginTx(c.session.Insert)
			if err != nil {
				c.err(err).Msg("Could not insert data into DB")
			}
		}()
	}
}

func (s *Server) pwCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	return nil, nil
}
