package server

import (
	"strings"

	"github.com/nightexcessive/excessiveircd/protocol"
	"github.com/sorcix/irc"
)

// CommandError is an error returned by a Command Func.
type CommandError struct {
	// Numeric is the numeric to use when sending the CommandError to the
	// Client.
	// If Numeric is an empty string, Param[0] is sent as a server notification
	// instead of a numeric.
	Numeric string
	Params  []string
}

// Command represents a command specification.
type Command struct {
	Func func(client *Client, message *irc.Message) *CommandError

	MinimumParams int

	Registered   bool
	Unregistered bool
}

var commands = map[string]*Command{
	irc.PASS: {cmdRegistration, 1, false, true},
	irc.NICK: {cmdRegistration, 1, true, true},
	irc.USER: {cmdRegistration, 4, false, true},
}

func cmdRegistration(c *Client, m *irc.Message) *CommandError {
	switch strings.ToLower(m.Command) {
	case "nick":
		nick := m.Params[0]
		if len(nick) <= 0 {
			nick = m.Trailing
		}

		if !protocol.IsValid(nick, protocol.Nickname) {
			return &CommandError{irc.ERR_ERRONEUSNICKNAME, []string{nick, "Erroneous nickname"}}
		}

		if c.Registered {
			reply := make(chan bool)
			c.Server.Events <- &SReregisterClient{nick, c, reply}
			if !<-reply {
				return &CommandError{irc.ERR_NICKNAMEINUSE, []string{nick, "Nickname is already in use"}}
			}
			c.writeMessage(&irc.Message{
				Prefix:   c.Info.Prefix,
				Command:  "NICK",
				Params:   nil,
				Trailing: nick,
			})
			c.Info.Name = nick
			return nil
		}

		c.Info.Name = nick
	case "user":
		user := m.Params[0]
		if len(user) <= 0 {
			user = m.Trailing
		}

		// TODO(nightexcessive): Should we be using a different validation set
		// here? The RFC doesn't say.
		if !protocol.IsValid(user, protocol.Nickname) {
			c.error("Invalid user name given")
			return nil
		}

		c.Info.User = "~" + user
	default:
		c.Logger.Printf("Unexpected registration command: %q", m.Command)
		return nil
	}

	if !c.Registered {
		if c.Info.Name != "*" && c.Info.User != "*" {
			reply := make(chan bool)
			c.Server.Events <- &SRegisterClient{c, reply}

			if !<-reply {
				return &CommandError{irc.ERR_NICKNAMEINUSE, []string{c.Info.Name, "Nickname is already in use"}}
			}

			c.Registered = true
			c.numeric(irc.RPL_WELCOME, "Welcome to the Internet Relay Network "+c.Info.String())
		}
		return nil
	}

	return nil
}
