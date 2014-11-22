package server

import (
	"strings"

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
		c.Info.Name = m.Params[0]
	case "user":
		c.Info.User = "~" + m.Params[0]
	default:
		c.Logger.Printf("Unexpected registration command: %q", m.Command)
		return nil
	}

	if !c.Registered && c.Info.Name != "*" && c.Info.User != "*" {

	}

	return nil
}
