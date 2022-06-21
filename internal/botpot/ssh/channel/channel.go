package channel

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

type Channel struct {
	start time.Time
	end   time.Time
	id    uint32
	p     *ssh.Client
	l     zerolog.Logger
}

// NewChannel creates a new channel
func NewChannel(id uint32, req ssh.NewChannel, proxy *ssh.Client, l zerolog.Logger) *Channel {
	ch := &Channel{
		id:    id,
		p:     proxy,
		l:     l.With().Uint32("chID", id).Logger(),
		start: time.Now(),
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
		c.l.Err(err).Msg("Could not open channel")
		err = chanReq.Reject(ssh.ConnectionFailed, "")
		if err != nil {
			c.l.Err(err).Msg("Could not reject channel request")
		}
		return
	}

	clientChan, clientReqChan, err := chanReq.Accept()
	if err != nil {
		c.l.Err(err).Msg("Could not accept channel request")
	}

	c.proxyChannelData(clientChan, proxyChan)           // handle the new channel
	go c.handleRequest(proxyChan, clientReqChan, true)  // client to proxy
	go c.handleRequest(clientChan, proxyReqChan, false) // proxy to client
}

// proxyChannelData proxies tdata between two SSH channels
func (c *Channel) proxyChannelData(clientChan, proxyChan ssh.Channel) {
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
			c.l.Err(err).Bool("fromClient", fromClient).Msg("Failed to copy")
		} else {
			c.l.Debug().Bool("fromClient", fromClient).Int64("bytesRead", n).Send()
		}
		if n > 0 {
			c.l.Debug().Bool("fromClient", fromClient).Str("data", string(buf.Bytes())).Send()
		}
	}

	go proxyFunc(clientChan, proxyChan, clientChan.Close, proxyChan.Close, true)
	go proxyFunc(clientChan.Stderr(), proxyChan.Stderr(), clientChan.Close, proxyChan.Close, true)
	go proxyFunc(proxyChan, clientChan, proxyChan.Close, clientChan.Close, false)
	go proxyFunc(proxyChan.Stderr(), clientChan.Stderr(), proxyChan.Close, clientChan.Close, false)
}

// handleRequest proxies requests between an SSH server and an SSH client
func (c *Channel) handleRequest(channel ssh.Channel, reqChan <-chan *ssh.Request, fromClient bool) {
	for req := range reqChan {
		c.logRequest(req, fromClient)
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
	INSERT INTO Channel(id, session_id, start_ts, end_ts)
		SELECT $1, MAX(Session.id), $2, $3
			FROM Session
`, c.id, c.start, c.end)
	return err
}

func (c *Channel) logRequest(req *ssh.Request, fromClient bool) {
	switch req.Type {
	case "pty-req":
		break
		//todo
	default:
		c.l.Info().Bool("fromClient", fromClient).Str("type", req.Type).Str("payload", string(req.Payload)).Bool("wantReply", req.WantReply).Msg("Received request")
	}
}
