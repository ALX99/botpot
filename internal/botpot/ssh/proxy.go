package ssh

import (
	"time"

	"golang.org/x/crypto/ssh"
)

// sshProxy represents an SSH connection where you can
// proxy stuff from the client to
type sshProxy struct {
	cfg     *ssh.ClientConfig
	client  *ssh.Client
	session *ssh.Session
	host    string
}

func newSSHProxy(host, user string) sshProxy {
	p := sshProxy{host: host}

	p.cfg = &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{}, // No auth
	}
	return p
}

// Connect connects to the SSH server with a backoff
func (p *sshProxy) Connect() error {
	var err error
	connect := func() error {
		p.client, err = ssh.Dial("tcp", p.host, p.cfg)
		if err != nil {
			return err
		}

		p.session, err = p.client.NewSession()
		return err
	}

	// Try to connect for 10s
	for i := 0; i < 100; i++ {
		err = connect()
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}

	return err
}

// Wait blocks until the connection has shut down, and returns the
// error causing the shutdown.
func (p *sshProxy) Wait() error {
	return p.client.Wait()
}

// Disconnect disconnects from the SSH server
func (p *sshProxy) Disconnect() error {
	err1 := p.session.Close()
	err2 := p.client.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
