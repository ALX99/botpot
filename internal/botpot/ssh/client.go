package ssh

import (
	"errors"
	"fmt"
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

type client struct {
	p            proxy
	conn         ssh.Conn
	channelchan  <-chan ssh.NewChannel
	name         string
	disconnected bool
	onDisconnect func()
	info         func() *zerolog.Event
	debug        func() *zerolog.Event
	err          func(err error) *zerolog.Event
}

// handle handles the client, and is non blocking
func (c *client) handle(reqChan <-chan *ssh.Request) {
	c.name = fmt.Sprintf("%s %s", c.conn.RemoteAddr(), c.conn.ClientVersion())
	// Setup loggers
	c.info = func() *zerolog.Event { return log.Info().Str("client", c.name) }
	c.debug = func() *zerolog.Event { return log.Debug().Str("client", c.name) }
	c.err = func(err error) *zerolog.Event { return log.Err(err).Str("client", c.name) }

	c.info().Msg("Connected")

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
	proxyFunc := func(read, write func([]byte) (int, error), rclose, wclose func() error, fromClient bool) {
		for !c.disconnected {
			b := make([]byte, 1024)
			i, err := read(b)
			if err != nil {
				if errors.Is(err, io.EOF) {
					c.debug().Bool("fromClient", fromClient).Int("bytesRead", i).Msg("EOF read")
					wclose()
					return
				}
				c.err(err).Bool("fromClient", fromClient).Msg("Failed to read")
				continue
			}

			i, err = write(b)
			if err != nil {
				if errors.Is(err, io.EOF) {
					c.debug().Bool("fromClient", fromClient).Int("bytesRead", i).Msg("EOF read")
					rclose()
					return
				}
				c.err(err).Bool("fromClient", fromClient).Msg("Failed to read")
			}
		}
	}
	go proxyFunc(clientChan.Read, proxyChan.Write, clientChan.Close, proxyChan.Close, true)
	go proxyFunc(clientChan.Stderr().Read, proxyChan.Stderr().Write, clientChan.Close, proxyChan.Close, true)
	go proxyFunc(proxyChan.Read, clientChan.Write, proxyChan.Close, clientChan.Close, false)
	go proxyFunc(proxyChan.Stderr().Read, clientChan.Stderr().Write, proxyChan.Close, clientChan.Close, false)
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
			continue
		}
		if req.WantReply {
			err = req.Reply(res, nil)
			if err != nil {
				c.err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
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
