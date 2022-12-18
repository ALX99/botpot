package ssh

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

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
	proxy        sshProxy
	l            zerolog.Logger
	session      session.Session
	chanCounter  uint32
	disconnected atomic.Bool
	wg           sync.WaitGroup
}

func newClient(conn ssh.Conn, proxy sshProxy, channelChan <-chan ssh.NewChannel) *client {
	l := log.With().
		Str("version", string(conn.ClientVersion())).
		Str("rAddr", conn.RemoteAddr().String()).
		Logger()

	s := session.NewSession(conn.RemoteAddr(), conn.LocalAddr(), string(conn.ClientVersion()), l)
	c := client{
		conn:         conn,
		rAddr:        conn.RemoteAddr(),
		channelchan:  channelChan,
		proxy:        proxy,
		l:            l,
		session:      s,
		chanCounter:  0,
		disconnected: atomic.Bool{},
		wg:           sync.WaitGroup{},
	}
	c.disconnected.Store(false)

	l.Info().Msg("Connected")
	return &c
}

// handle handles the client and blocks until client has disconnected
func (c *client) handle(reqChan <-chan *ssh.Request) {
	t := time.Now()
	err := c.proxy.Connect()
	if err != nil {
		c.l.Err(err).Msg("Could not connect to proxy")
		c.conn.Close()
		return
	}

	c.l.Info().Str("duration", time.Since(t).String()).Msg("Connected to proxy")
	c.wg.Add(2)
	go c.handleChannels()
	go c.handleGlobalRequests(c.proxy.client, reqChan, true) // client to proxy

	// Wait for proxy to disconnect
	go func() {
		c.proxy.Wait()
		if c.disconnected.Load() {
			return // client already disconnected
		}
		c.l.Err(errors.New("proxy disconnected without client")).Msg("Something went wrong")

		// Disconnect client, something has gone wrong
		if err = c.conn.Close(); err != nil {
			c.l.Err(err).Msg("Error while disconnecting client")
		}
		c.disconnected.Store(true)
	}()

	// Wait for client to disconnect
	c.conn.Wait()
	c.disconnected.Store(true)

	c.session.Stop()

	err = c.proxy.Disconnect()
	if err != nil {
		c.l.Err(err).Msg("Error while disconnecting proxy")
	}
	c.wg.Wait()
}

// handleChannels handles channel requests from the client
func (c *client) handleChannels() {
	for chanReq := range c.channelchan {
		ch := channel.NewChannel(atomic.AddUint32(&c.chanCounter, 1), chanReq, c.proxy.client, c.l)
		ch.Handle()
		c.session.AddChannel(ch)
	}
	c.wg.Done()
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
	c.wg.Done()
}
