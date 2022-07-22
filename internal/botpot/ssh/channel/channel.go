package channel

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alx99/botpot/internal/botpot/sftp"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

// Channel represents an SSH channel
type Channel struct {
	start       time.Time
	end         time.Time
	p           *ssh.Client
	proxyClosed *int32
	recvStderr  *bytes.Buffer
	recv        *bytes.Buffer
	closed      *int32
	channelType string
	l           zerolog.Logger
	reqs        []request
	id          uint32
}

// NewChannel creates a new channel
func NewChannel(id uint32, req ssh.NewChannel, proxy *ssh.Client, l zerolog.Logger) *Channel {
	ch := &Channel{
		start:       time.Now(),
		end:         time.Time{},
		p:           proxy,
		proxyClosed: new(int32),
		recvStderr:  &bytes.Buffer{},
		recv:        &bytes.Buffer{},
		channelType: req.ChannelType(),
		l:           l.With().Uint32("chID", id).Logger(),
		reqs:        []request{},
		id:          id,
		closed:      new(int32),
	}

	ch.handle(req)

	return ch
}

// handle accepts te new channel requests and starts
// necessary goroutines
func (c *Channel) handle(chanReq ssh.NewChannel) {
	c.l.Info().Str("type", chanReq.ChannelType()).Str("extraData", string(chanReq.ExtraData())).Msg("Wants to open channel")

	proxyChan, proxyReqChan, err := c.p.OpenChannel(chanReq.ChannelType(), chanReq.ExtraData())
	if err != nil {
		c.end = time.Now() // ensure endtime is not null value in case of error
		c.l.Err(err).Msg("Could not open channel")
		err = chanReq.Reject(ssh.ConnectionFailed, "")
		if err != nil {
			c.l.Err(err).Msg("Could not reject channel request")
		}
		return
	}

	clientChan, clientReqChan, err := chanReq.Accept()
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
	clientClosed := new(int32)
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
				atomic.AddInt32(c.proxyClosed, 1)
			} else {
				if atomic.AddInt32(clientClosed, 1) == 2 && atomic.LoadInt32(c.closed) != 1 {
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
				err = req.Reply(false, nil)
				if err != nil {
					c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
		} else {
			if req.WantReply {
				err = req.Reply(res, nil)
				if err != nil {
					c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to reply to request")
				}
			}
		}
	}

	// Here we know there will be no new requests from the proxy
	if !fromClient {
		// We will take care of closing the client channel
		atomic.AddInt32(c.closed, 1)

		// This allows us to wait until all channel data
		// from the proxy has been read and sent to the client channel.
		for atomic.LoadInt32(c.proxyClosed) != 2 {
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
		err = req.Insert(tx)
		if err != nil {
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
		parser := sftp.NewParser(c.recv.Bytes(), c.l)
		err = parser.Parse()
		if err != nil {
			return err
		}
	}

	return nil
}
