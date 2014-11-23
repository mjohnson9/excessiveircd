package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/sorcix/irc"

	"code.google.com/p/go-uuid/uuid"
)

// Client represents a client connection.
type Client struct {
	ID     uuid.UUID
	Server *Server
	Logger *log.Logger

	Info struct {
		*irc.Prefix
		Real string
	}

	Registered bool
	Closed     bool

	IP net.IP

	Events chan interface{}

	conn net.Conn
	buf  *bufio.Reader

	*sync.RWMutex
}

// NewClient creates and initializes a new Client. Once done initializing, it
// registers the new client with the given Server.
func NewClient(conn net.Conn, server *Server) *Client {
	id := uuid.NewRandom()
	client := &Client{
		ID:     id,
		Server: server,
		Logger: log.New(os.Stderr, fmt.Sprintf("Client(%s) ", id), 0),

		Info: struct {
			*irc.Prefix
			Real string
		}{
			&irc.Prefix{
				Name: "*",
				User: "*",
				Host: "*",
			},
			"*",
		},

		Events: make(chan interface{}),

		conn: conn,
		buf:  bufio.NewReaderSize(conn, 512), // 512 byte buffer as per RFC1459

		RWMutex: new(sync.RWMutex),
	}

	go client.eventLoop()
	go client.readLoop()

	client.Events <- new(CInitialize)

	return client
}

func (c *Client) lookupHostname() {
	c.serverNotice(c.Server, "*** Looking up your hostname...")

	ipRaw, _, _ := net.SplitHostPort(c.conn.RemoteAddr().String())
	c.IP = net.ParseIP(ipRaw)
	if c.IP == nil {
		c.Logger.Printf("Failed to parse IP: %q", ipRaw)
		c.close("Error looking up hostname")
		return
	}

	names, err := net.LookupAddr(c.IP.String())
	if err != nil {
		c.Logger.Printf("Error in LookupAddr: %s", err)
		c.Info.Host = c.IP.String()
		c.serverNotice(c.Server, "*** Could not find your hostname.")
		return
	}

	for _, hostname := range names {
		ips, err := net.LookupIP(hostname)
		if err != nil {
			continue
		}

		for _, resolvedIP := range ips {
			if c.IP.Equal(resolvedIP) {
				c.Info.Host = hostname
				c.serverNotice(c.Server, "*** Found your hostname: "+c.Info.Host)
				return
			}
		}
	}

	c.Info.Host = c.IP.String()
	c.serverNotice(c.Server, "*** Could not find your hostname.")
}

func (c *Client) readLoop() {
	for {
		line, err := c.buf.ReadString('\n')
		if netErr, ok := err.(net.Error); ok {
			if !netErr.Temporary() {
				c.Logger.Printf("Read error (net.Error, non-temporary): %s", err)
				c.Events <- &CClose{"Read error: " + err.Error()}
				return
			}
			c.Logger.Printf("Read error (net.Error, temporary): %s", err)
		} else if err == io.EOF || (err != nil && strings.HasSuffix(err.Error(), "use of closed network connection")) {
			c.Events <- &CClose{"Connection reset by peer"}
			return
		} else if err != nil {
			c.Logger.Printf("Read error: %s", err)
			c.Events <- &CClose{"Read error: " + err.Error()}
			return
		}
		// Remove LF
		line = line[:len(line)-1]
		// Remove CR
		if len(line) >= 1 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		c.handleLine(line)
	}
}

func (c *Client) eventLoop() {
	for event := range c.Events {
		c.Logger.Printf("Event: %T", event)
		switch ev := event.(type) {
		case *CInitialize:
			c.lookupHostname()
		case *CMessage:
			c.handleMessage(ev.Message)
		case *CClose:
			c.close(ev.Reason)
		default:
			c.Logger.Printf("Unexpected event of type %T: %#v", ev, ev)
		}
	}
}

func (c *Client) handleLine(line string) {
	message := irc.ParseMessage(line)
	if message == nil {
		c.Logger.Printf("Error in parsing %q", line)
		return
	}

	c.Events <- &CMessage{message}
}

func (c *Client) handleMessage(m *irc.Message) {
	lowerCmd := strings.ToUpper(m.Command)
	commandEntry, found := commands[lowerCmd]
	if !found {
		c.numeric(irc.ERR_UNKNOWNCOMMAND, m.Command, "Unknown command")
		return
	}

	numParams := len(m.Params)
	if len(m.Trailing) > 0 {
		numParams++
	}
	if numParams < commandEntry.MinimumParams {
		c.numeric(irc.ERR_NEEDMOREPARAMS, m.Command, "Not enough parameters")
		return
	}

	if c.Registered && !commandEntry.Registered {
		c.numeric(irc.ERR_ALREADYREGISTRED, "Unauthorized command (already registered)")
		return
	}

	if !c.Registered && !commandEntry.Unregistered {
		c.numeric(irc.ERR_NOTREGISTERED, "You have not registered")
		return
	}

	err := commandEntry.Func(c, m)
	if err == nil {
		return
	}

	if err.Numeric == "" {
		c.serverNotice(c.Server, err.Params[0])
		return
	}

	c.numeric(err.Numeric, err.Params...)
}

func (c *Client) writeString(line string) (int, error) {
	return io.WriteString(c.conn, line+"\r\n")
}

func (c *Client) writeMessage(m *irc.Message) (int, error) {
	return io.WriteString(c.conn, m.String()+"\r\n")
}

func (c *Client) close(reason string) {
	close(c.Events)
	c.error("Closing link " + c.Info.Name + ": " + reason)
	c.conn.Close()

	reply := make(chan struct{})
	c.Server.Events <- &SDeregisterClient{c, reply}
	<-reply
}

func (c *Client) error(text string) {
	c.writeString("ERROR :" + text)
}

func (c *Client) serverNotice(s *Server, text string) {
	c.writeString(":" + s.FriendlyName() + " NOTICE " + c.Info.Name + " :" + text)
}

func (c *Client) numeric(numeric string, args ...string) {
	if len(args) > 0 {
		lastNum := len(args) - 1
		lastArg := args[lastNum]
		args[lastNum] = ":" + lastArg
	}

	c.rawNumeric(numeric, args...)
}

func (c *Client) rawNumeric(numeric string, args ...string) {
	c.writeString(fmt.Sprintf(":%s %s %s %s", c.Server.FriendlyName(), numeric, c.Info.Name, strings.Join(args, " ")))
}
