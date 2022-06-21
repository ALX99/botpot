package ssh

import (
	"errors"
	"net"
	"sync/atomic"

	"github.com/alx99/botpot/internal/botpot/ssh/channel"
	"github.com/alx99/botpot/internal/botpot/ssh/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

type client struct {
	conn         ssh.Conn
	rAddr        net.Addr
	channelchan  <-chan ssh.NewChannel
	p            sshProxy
	l            zerolog.Logger
	s            session.Session
	chanCounter  uint32
	disconnected bool
}

func newClient(conn ssh.Conn, p sshProxy, channelChan <-chan ssh.NewChannel) client {
	l := log.With().
		Str("version", string(conn.ClientVersion())).
		Str("rAddr", conn.RemoteAddr().String()).
		Logger()
	s := session.NewSession(conn.RemoteAddr(), conn.LocalAddr(), string(conn.ClientVersion()), l)
	c := client{
		p:           p,
		conn:        conn,
		channelchan: channelChan,
		rAddr:       conn.RemoteAddr(),
		s:           s,
		l:           l,
	}

	l.Info().Msg("Connected")
	return c
}

// handle handles the client and blocks until client has disconnected
func (c *client) handle(reqChan <-chan *ssh.Request) {
	err := c.p.Connect()
	if err != nil {
		c.l.Err(err).Msg("Could not connect to proxy")
		c.conn.Close()
		return
	}

	c.l.Info().Msg("Connected to proxy")
	go c.handleChannels()
	go c.handleGlobalRequests(c.p.client, reqChan, true) // client to proxy

	// Wait for proxy to disconnect
	go func() {
		c.p.Wait()
		if c.disconnected {
			return
		}
		c.l.Err(errors.New("proxy disconnected without client")).Msg("Something went wrong")

		// Disconnect client, something has gone wrong
		err := c.conn.Close()
		if err != nil {
			c.l.Err(err).Msg("Error while disconnecting client")
		}
		c.disconnected = true
	}()

	// Wait for client to disconnect
	c.conn.Wait()
	c.disconnected = true

	c.s.Stop()

	err = c.p.Disconnect()
	if err != nil {
		c.l.Err(err).Msg("Error while disconnecting proxy")
	}
}

// handleChannels handles channel requests from the client
func (c *client) handleChannels() {
	for chanReq := range c.channelchan {
		// todo
		channel.NewChannel(atomic.AddUint32(&c.chanCounter, 1), chanReq, c.p.client, c.l)
	}
}

// handleGlobalRequests proxies global requests from the client to an SSH server
func (c *client) handleGlobalRequests(client *ssh.Client, reqChan <-chan *ssh.Request, fromClient bool) {
	for req := range reqChan {
		// This we actually ignore because this will give us
		// side-effects if our proxy respects this request
		if req.Type == "no-more-sessions@openssh.com" {
			continue
		}

		// todo
		// c.logRequest(req, fromClient)

		ok, res, err := client.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to proxy request")
			if req.WantReply {
				err = req.Reply(false, res)
				if err != nil {
					c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
			continue
		}

		if req.WantReply {
			err = req.Reply(ok, res)
			if err != nil {
				c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
			}
		}
	}
}
