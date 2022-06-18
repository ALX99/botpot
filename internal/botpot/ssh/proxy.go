package ssh

import (
	"time"

	"golang.org/x/crypto/ssh"
)

type proxy struct {
	cfg     *ssh.ClientConfig
	client  *ssh.Client
	session *ssh.Session
	host    string
}

func newProxy(host, user, password string) proxy {
	p := proxy{host: host}

	p.cfg = &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}
	return p
}

// Connect connects to the SSH server with a backoff
func (p *proxy) Connect() error {
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
func (p *proxy) Wait() error {
	return p.client.Wait()
}

func (p *proxy) openChannel(name string, data []byte) (ssh.Channel, <-chan *ssh.Request, error) {
	return p.client.OpenChannel(name, data)
}

func (p *proxy) sendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return p.session.SendRequest(name, wantReply, payload)
}

// Disconnect disconnects from the SSH server
func (p *proxy) Disconnect() error {
	err1 := p.session.Close()
	err2 := p.client.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
