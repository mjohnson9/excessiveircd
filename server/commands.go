package server

import (
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
	//irc.PASS: {cmdRegistration, 1, false, true},
	irc.NICK: {cmdRegistration, 1, true, true},
	irc.USER: {cmdRegistration, 4, false, true},
}

// cmdChangeNick is called when an already registered user uses the NICK command.
func cmdChangeNick(c *Client, m *irc.Message, nick string) *CommandError {
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

func cmdRegistration(c *Client, m *irc.Message) *CommandError {
	switch m.Command {
	case irc.NICK:
		nick := m.Params[0]

		if !protocol.IsValid(nick, protocol.Nickname) {
			return &CommandError{irc.ERR_ERRONEUSNICKNAME, []string{nick, "Erroneous nickname"}}
		}

		if c.Registered {
			return cmdChangeNick(c, m, nick)
		}

		c.Info.Name = nick
	case irc.USER:
		user := m.Params[0]

		if !protocol.IsValid(user, protocol.Username) {
			c.error("Invalid user name given")
			return nil
		}

		c.Info.User = "~" + user
	default:
		c.Logger.Printf("Unexpected registration command: %q", m.Command)
		return nil
	}

	// Registered connections should never reach this section.

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
