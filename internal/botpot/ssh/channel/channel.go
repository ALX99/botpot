package channel

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

// Channel represents an SSH channel
type Channel struct {
	start       time.Time
	end         time.Time
	p           *ssh.Client
	closed      *int32
	channelType string
	l           zerolog.Logger
	reqs        []request
	id          uint32

	// Data received by the client
	recvStderr *bytes.Buffer
	recv       *bytes.Buffer
}

// NewChannel creates a new channel
func NewChannel(id uint32, req ssh.NewChannel, proxy *ssh.Client, l zerolog.Logger) *Channel {
	ch := &Channel{
		start:       time.Now(),
		end:         time.Time{},
		p:           proxy,
		channelType: req.ChannelType(),
		l:           l.With().Uint32("chID", id).Logger(),
		reqs:        []request{},
		id:          id,
		closed:      new(int32),
		recv:        &bytes.Buffer{},
		recvStderr:  &bytes.Buffer{},
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
	proxyFunc := func(read io.Reader, write io.Writer, fromClient bool) {
		n, err := io.Copy(write, read)
		if err != nil {
			c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to copy")
		} else {
			c.l.Debug().Bool("fromClient", fromClient).Int64("bytesRead", n).Send()

			// Keep track of when the proxy has closed
			// the channel connection
			if !fromClient {
				atomic.AddInt32(c.closed, 1)
			}
		}
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

		// https://datatracker.ietf.org/doc/html/rfc4254#section-6.10
		if req.Type == "exit-status" && !fromClient {
			// This allows us to wait until all channel data
			// from the proxy has been read. That means that there
			// will not be any more data needed to be sent over the channel
			for atomic.LoadInt32(c.closed) != 2 {
				time.Sleep(10 * time.Millisecond)
			}

			c.l.Info().Bool("fromClient", fromClient).Msg("Done")
			if err = channel.Close(); err != nil {
				c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to close channel")
			}
		}
	}
	// If we reach this point it means that the reqChan
	// client has been closed and thus the channel
	// TODO read RFC
	if fromClient {
		c.end = time.Now()
	}

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

	for _, req := range c.reqs {
		err = req.Insert(tx)
		if err != nil {
			return err
		}
	}

	return nil
}
