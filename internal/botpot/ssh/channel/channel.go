package channel

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alx99/botpot/internal/botpot/sftp"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

// Channel represents an SSH channel
type Channel struct {
	start        time.Time
	end          time.Time
	reqChan      ssh.NewChannel
	p            *ssh.Client
	recvStderr   *bytes.Buffer
	recv         *bytes.Buffer
	proxyClosed  atomic.Bool // true if the proxy channel has been closed
	clientClosed atomic.Bool // true if the channel has been closed
	channelType  string
	l            zerolog.Logger
	reqs         []request
	id           uint32
}

// NewChannel creates a new channel
func NewChannel(id uint32, req ssh.NewChannel, proxy *ssh.Client, l zerolog.Logger) *Channel {
	ch := &Channel{
		start:        time.Now(),
		end:          time.Time{},
		reqChan:      req,
		p:            proxy,
		proxyClosed:  atomic.Bool{},
		recvStderr:   &bytes.Buffer{},
		recv:         &bytes.Buffer{},
		clientClosed: atomic.Bool{},
		channelType:  req.ChannelType(),
		l:            l.With().Uint32("chID", id).Logger(),
		reqs:         []request{},
		id:           id,
	}

	return ch
}

// Handle starts handleling the channel and
// accepts new channel requests
func (c *Channel) Handle() {
	c.l.Info().Str("type", c.reqChan.ChannelType()).Str("extraData", string(c.reqChan.ExtraData())).Msg("Wants to open channel")

	proxyChan, proxyReqChan, err := c.p.OpenChannel(c.reqChan.ChannelType(), c.reqChan.ExtraData())
	if err != nil {
		c.end = time.Now() // ensure endtime is not null value in case of error
		c.l.Err(err).Msg("Could not open channel")
		if err = c.reqChan.Reject(ssh.ConnectionFailed, ""); err != nil {
			c.l.Err(err).Msg("Could not reject channel request")
		}
		return
	}

	clientChan, clientReqChan, err := c.reqChan.Accept()
	if err != nil {
		c.end = time.Now() // ensure endtime is not null value in case of error
		c.l.Err(err).Msg("Could not accept channel request")
		return
	}

	c.proxyChannelData(clientChan, proxyChan)           // handle the new channel
	go c.handleRequest(proxyChan, clientReqChan, true)  // client to proxy
	go c.handleRequest(clientChan, proxyReqChan, false) // proxy to client
}

// proxyChannelData proxies data between two SSH channels
func (c *Channel) proxyChannelData(clientChan, proxyChan ssh.Channel) {
	clientClosed := atomic.Bool{}
	proxyFunc := func(read io.Reader, write io.Writer, fromClient bool) {
		n, err := io.Copy(write, read)
		if err != nil {
			c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to copy")
		} else {
			// Keep track of when the proxy has closed
			// the channel connection
			// This is needed since we want to wait with closing the client
			// SSH channel until all data has been sent
			if !fromClient {
				c.proxyClosed.Store(true)
			} else {
				if !clientClosed.Swap(true) && !c.clientClosed.Load() {
					// Here the client has left us without the proxy request channel being closed
					// This case can for example be hit when dealing with SFTP
					// Time to bail
					if err = proxyChan.Close(); err != nil {
						c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to close channel")
					}
				}
			}
		}
		c.l.Debug().Bool("fromClient", fromClient).Int64("bytesRead", n).Send()
	}

	go proxyFunc(io.TeeReader(clientChan, c.recv), proxyChan, true)
	go proxyFunc(io.TeeReader(clientChan.Stderr(), c.recvStderr), proxyChan.Stderr(), true)
	go proxyFunc(proxyChan, clientChan, false)
	go proxyFunc(proxyChan.Stderr(), clientChan.Stderr(), false)
}

// handleRequest proxies requests between an SSH server and an SSH client
func (c *Channel) handleRequest(channel ssh.Channel, reqChan <-chan *ssh.Request, fromClient bool) {
	for req := range reqChan {
		parsedReq, err := newRequest(req, fromClient, c.id, c.l.With().Bool("fromClient", fromClient).Logger())
		if err != nil {
			c.l.Err(err).Bool("fromClient", fromClient).Msg("Error while creating a new Request")
		} else {
			c.reqs = append(c.reqs, parsedReq)
		}

		res, err := channel.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to proxy request")
			if req.WantReply {
				if err = req.Reply(false, nil); err != nil {
					c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
		} else {
			if req.WantReply {
				if err = req.Reply(res, nil); err != nil {
					c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
		}
	}

	// Here we know there will be no new requests from the proxy
	if !fromClient {
		// We will take care of closing the client channel
		c.clientClosed.Store(true)

		// Wait until all channel data from the proxy
		// has been read and sent to the client channel.
		for !c.proxyClosed.Load() {
			time.Sleep(10 * time.Millisecond)
		}

		if err := channel.Close(); err != nil {
			c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to close channel")
		}
	}

	// If we reach this point it means that the reqChan
	// client has been closed and thus the channel
	// TODO read RFC
	if fromClient {
		c.end = time.Now()
	}

	c.l.Info().Bool("fromClient", fromClient).Msg("All requests served")
}

// Insert tries to insert the data into the database
func (c *Channel) Insert(tx pgx.Tx) error {
	_, err := tx.Exec(context.TODO(), `
	INSERT INTO Channel(id, session_id, channel_type, recv, recv_stderr, start_ts, end_ts)
		SELECT $1, MAX(Session.id), $2, $3, $4, $5, $6
			FROM Session
`, c.id, c.channelType, c.recv.Bytes(), c.recvStderr.Bytes(), c.start, c.end)
	if err != nil {
		return err
	}

	sftpFound := false
	for _, req := range c.reqs {
		if err = req.Insert(tx); err != nil {
			return err
		}

		// Look for SFTP subsystem
		switch v := req.(type) {
		case *subSystemRequest:
			if strings.ToLower(v.Name) == "sftp" {
				sftpFound = true
			}
		default:
		}
	}

	if sftpFound {
		// https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-13#section-3.1
		parser := sftp.NewParser(c.recv.Bytes(), c.l)
		if err = parser.Parse(); err != nil {
			return err
		}
	}

	return nil
}
