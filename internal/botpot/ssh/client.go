package ssh

import (
	"errors"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/ssh"
)

type client struct {
	p            proxy
	conn         ssh.Conn
	channelchan  <-chan ssh.NewChannel
	name         string
	disconnected bool
	onDisconnect func()
}

// handle handles the client, and is non blocking
func (c *client) handle(reqChan <-chan *ssh.Request) {
	c.name = fmt.Sprintf("%s %s", c.conn.RemoteAddr(), c.conn.ClientVersion())
	log.Println(c.name, "connected")

	err := c.p.Connect()
	if err != nil {
		log.Println(c.name, "could not connect to proxy", err)
		c.disconnect()
		return
	}

	log.Println(c.name, "connected to proxy")
	go c.handleChannels()
	go c.handleGlobalRequests(c.p.client, reqChan) // client to proxy

	go func() {
		c.conn.Wait()
		c.disconnected = true
		log.Println(c.name, "disconnected")
		err = c.p.Disconnect()
		if err != nil {
			log.Println(c.name, "error while disconnecting proxy", err)
		}
		c.onDisconnect()
	}()

	go func() {
		c.p.Wait()
		if c.disconnected {
			return
		}
		log.Println(c.name, "something went wrong, proxy was disconnected")
		err := c.disconnect()
		if err != nil {
			log.Println("client disconnect error", err)
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
		log.Println(c.name, "wants to open channel", chanReq.ChannelType(), string(chanReq.ExtraData()))

		proxyChan, proxyReqChan, err := c.p.openChannel(chanReq.ChannelType(), chanReq.ExtraData())
		if err != nil {
			log.Println(c.name, "could not open channel", err)
			err = chanReq.Reject(ssh.ConnectionFailed, "")
			if err != nil {
				log.Println(c.name, "could not reject channel request", err)
			}
			continue
		}
		clientChan, clientReqChan, err := chanReq.Accept()
		if err != nil {
			log.Println(c.name, "could not accept channel request", err)
		}

		go c.handleChannel(clientChan, proxyChan)           // handle the new channel
		go c.handleChannelRequest(proxyChan, clientReqChan) // client to proxy
		go c.handleChannelRequest(clientChan, proxyReqChan) // proxy to client
	}
}

// handleChannel handles a channel from a client
func (c *client) handleChannel(clientChan, proxyChan ssh.Channel) {
	proxyFunc := func(read, write func([]byte) (int, error), rclose, wclose func() error, interesting bool) {
		for !c.disconnected {
			b := make([]byte, 1024)
			i, err := read(b)
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Println(c.name, "EOF read", i)
					wclose()
					return
				}
				log.Println(c.name, "failed to read", err)
				continue
			}
			// todo log later
			// if interesting {
			// 	fmt.Print(string(b))
			// }
			i, err = write(b)
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Println(c.name, "EOF write", i)
					rclose()
					return
				}
				log.Println(c.name, "failed to write", err)
			}
		}
	}
	go proxyFunc(clientChan.Read, proxyChan.Write, clientChan.Close, proxyChan.Close, true)
	go proxyFunc(clientChan.Stderr().Read, proxyChan.Stderr().Write, clientChan.Close, proxyChan.Close, true)
	go proxyFunc(proxyChan.Read, clientChan.Write, proxyChan.Close, clientChan.Close, false)
	go proxyFunc(proxyChan.Stderr().Read, clientChan.Stderr().Write, proxyChan.Close, clientChan.Close, false)
}

// handleChannelRequest proxies requests between an SSH server and an SSH client
func (c *client) handleChannelRequest(channel ssh.Channel, reqChan <-chan *ssh.Request) {
	for req := range reqChan {
		c.logRequest(req)
		res, err := channel.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			log.Println(c.name, "failed to proxy request", err)
			if req.WantReply {
				err = req.Reply(false, nil)
				if err != nil {
					log.Println(c.name, "failed to reply", err)
				}
			}
			continue
		}
		if req.WantReply {
			err = req.Reply(res, nil)
			if err != nil {
				log.Println(c.name, "failed to reply", err)
			}
		}
	}
}
func (c *client) logRequest(req *ssh.Request) {
	switch req.Type {
	case "exec":
		log.Printf("%s exec %s\n", c.name, req.Payload)
		return

	}
	log.Println(c.name, "channel request", req.Type)
}

// handleGlobalRequests proxies global requests from the client to an SSH server
func (c *client) handleGlobalRequests(client *ssh.Client, reqChan <-chan *ssh.Request) {
	for req := range reqChan {
		log.Println(c.name, "request", req.Type)
		ok, res, err := client.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			log.Println(c.name, "failed to proxy request", err)
			if req.WantReply {
				err = req.Reply(false, res)
				if err != nil {
					log.Println(c.name, "failed to reply", err)
				}
			}
			continue
		}
		if req.WantReply {
			err = req.Reply(ok, res)
			if err != nil {
				log.Println(c.name, "failed to reply", err)
			}
		}
	}
}
