package ssh

import (
	"bytes"
	"errors"
	"io"
	"net"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

type client struct {
	conn         ssh.Conn
	info         func() *zerolog.Event
	channelchan  <-chan ssh.NewChannel
	onDisconnect func()
	debug        func() *zerolog.Event
	err          func(err error) *zerolog.Event
	p            sshProxy
	rAddr        net.Addr
	version      string
	disconnected bool
}

func newClient(conn ssh.Conn, p sshProxy, channelChan <-chan ssh.NewChannel, onDisconnect func()) client {
	c := client{
		p:            p,
		conn:         conn,
		channelchan:  channelChan,
		onDisconnect: onDisconnect,
		rAddr:        conn.RemoteAddr(),
		version:      string(conn.ClientVersion()),
	}

	// Setup loggers
	c.info = func() *zerolog.Event { return log.Info().Str("version", c.version).Str("rAddr", c.rAddr.String()) }
	c.debug = func() *zerolog.Event { return log.Debug().Str("version", c.version).Str("rAddr", c.rAddr.String()) }
	c.err = func(err error) *zerolog.Event {
		return log.Err(err).Str("version", c.version).Str("rAddr", c.rAddr.String())
	}

	c.info().Msg("Connected")
	return c
}

// handle handles the client, and is non blocking
func (c *client) handle(reqChan <-chan *ssh.Request) {
	err := c.p.Connect()
	if err != nil {
		c.err(err).Msg("Could not connect to proxy")
		c.disconnect()
		return
	}

	c.info().Msg("Connected to proxy")
	go c.handleChannels()
	go c.handleGlobalRequests(c.p.client, reqChan, true) // client to proxy

	go func() {
		c.conn.Wait()
		c.disconnected = true
		c.info().Msg("Disconnected")
		err = c.p.Disconnect()
		if err != nil {
			c.err(err).Msg("Error while disconnecting proxy")
		}
		c.onDisconnect()
	}()

	go func() {
		c.p.Wait()
		if c.disconnected {
			return
		}
		c.err(errors.New("proxy disconnected without client")).Msg("Something went wrong")
		err := c.disconnect()
		if err != nil {
			c.err(err).Msg("Error while disconnecting client")
		}
		c.disconnected = true
	}()
}

func (c *client) disconnect() error {
	return c.conn.Close()
}

// handleChannels handles channel requests from the client
func (c *client) handleChannels() {
	for chanReq := range c.channelchan {
		c.info().Str("type", chanReq.ChannelType()).Str("extraData", string(chanReq.ExtraData())).Msg("Wants to open channel")

		proxyChan, proxyReqChan, err := c.p.openChannel(chanReq.ChannelType(), chanReq.ExtraData())
		if err != nil {
			c.err(err).Msg("Could not open channel")
			err = chanReq.Reject(ssh.ConnectionFailed, "")
			if err != nil {
				c.err(err).Msg("Could not reject channel request")
			}
			continue
		}
		clientChan, clientReqChan, err := chanReq.Accept()
		if err != nil {
			c.err(err).Msg("Could not accept channel request")
		}

		go c.handleChannel(clientChan, proxyChan)                  // handle the new channel
		go c.handleChannelRequest(proxyChan, clientReqChan, true)  // client to proxy
		go c.handleChannelRequest(clientChan, proxyReqChan, false) // proxy to client
	}
}

// handleChannel handles a channel from a client
func (c *client) handleChannel(clientChan, proxyChan ssh.Channel) {
	proxyFunc := func(read io.Reader, write io.Writer, rclose, wclose func() error, fromClient bool) {
		var buf bytes.Buffer
		read = io.TeeReader(read, &buf)
		n, err := io.Copy(write, read)
		if err != nil {
			// Try to close both ios if we get an EOF error
			// and ignore errors
			if errors.Is(err, io.EOF) {
				rclose()
				wclose()
			}
			c.err(err).Bool("fromClient", fromClient).Msg("Failed to copy")
		} else {
			c.debug().Bool("fromClient", fromClient).Int64("bytesRead", n).Send()
		}
		if n > 0 {
			c.debug().Bool("fromClient", fromClient).Str("data", string(buf.Bytes())).Send()
		}
	}
	go proxyFunc(clientChan, proxyChan, clientChan.Close, proxyChan.Close, true)
	go proxyFunc(clientChan.Stderr(), proxyChan.Stderr(), clientChan.Close, proxyChan.Close, true)
	go proxyFunc(proxyChan, clientChan, proxyChan.Close, clientChan.Close, false)
	go proxyFunc(proxyChan.Stderr(), clientChan.Stderr(), proxyChan.Close, clientChan.Close, false)
}

// handleChannelRequest proxies requests between an SSH server and an SSH client
func (c *client) handleChannelRequest(channel ssh.Channel, reqChan <-chan *ssh.Request, fromClient bool) {
	for req := range reqChan {
		c.logRequest(req, fromClient)
		res, err := channel.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			c.err(err).Bool("fromClient", fromClient).Msg("Failed to proxy request")
			if req.WantReply {
				err = req.Reply(false, nil)
				if err != nil {
					c.err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
		} else {
			if req.WantReply {
				err = req.Reply(res, nil)
				if err != nil {
					c.err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
		}
		// https://datatracker.ietf.org/doc/html/rfc4254#section-6.10
		if req.Type == "exit-status" && !fromClient {
			if err = channel.Close(); err != nil {
				c.err(err).Bool("fromClient", fromClient).Msg("Failed to close channel")
			}
		}
	}
}

// handleGlobalRequests proxies global requests from the client to an SSH server
func (c *client) handleGlobalRequests(client *ssh.Client, reqChan <-chan *ssh.Request, fromClient bool) {
	for req := range reqChan {
		c.logRequest(req, fromClient)
		ok, res, err := client.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			c.err(err).Bool("fromClient", fromClient).Msg("Failed to proxy request")
			if req.WantReply {
				err = req.Reply(false, res)
				if err != nil {
					c.err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
			continue
		}
		if req.WantReply {
			err = req.Reply(ok, res)
			if err != nil {
				c.err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
			}
		}
	}
}

func (c *client) logRequest(req *ssh.Request, fromClient bool) {
	switch req.Type {
	case "pty-req":
		break
		//todo
	default:
		c.info().Bool("fromClient", fromClient).Str("type", req.Type).Str("payload", string(req.Payload)).Bool("wantReply", req.WantReply).Msg("Received request")
	}
}
