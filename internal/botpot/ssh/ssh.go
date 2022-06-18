package ssh

import (
	"io/ioutil"
	"log"
	"net"
	"strconv"

	"github.com/alx99/botpot/internal/hostprovider"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	l         net.Listener
	provider  hostprovider.SSH
	cfg       *ssh.ServerConfig
	port      int
	lIsClosed bool
}

// New creates a new SSH server
func New(port int, provider hostprovider.SSH) *Server {
	s := &Server{
		port:     port,
		provider: provider,
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
	hostKey, err := readHostKey("./key")
	if err != nil {
		return err
	}
	s.cfg.AddHostKey(hostKey)

	s.l, err = net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return err
	}
	log.Println("Started listening")

	go s.loop()
	return nil
}

// Stop stops the SSH server
func (s *Server) Stop() error {
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
			log.Println("Could not accept connection,", err)
			continue
		}
		// Handshake connection
		sshConn, channelChan, reqChan, err := ssh.NewServerConn(conn, s.cfg)
		if err != nil {
			log.Println("Could not handshake SSH connection,", err)
			conn.Close()
			continue
		}

		host, ID, err := s.provider.GetHost()
		if err != nil {
			log.Println("Could not get a hold of an SSH host", err)
			continue
		}
		p := newProxy(host, "panda", "password")
		// Handle new client
		c := client{
			p:           p,
			conn:        sshConn,
			channelchan: channelChan,
			onDisconnect: func() {
				err := s.provider.StopHost(ID)
				if err != nil {
					log.Println("onDisconnect failed", err)
				}
			},
		}
		c.handle(reqChan)
	}
}

func (s *Server) pwCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	return nil, nil
}
