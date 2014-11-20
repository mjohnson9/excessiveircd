package server

import (
	"io"
	"strings"
)

// Message represents an IRC message line.
type Message struct {
	Prefix  string
	Command string
	Params  []string
}

func encodeParam(param string) string {
	if strings.Contains(param, " ") {
		return ":" + param
	}

	return param
}

func (m *Message) String() string {
	output := ""

	if len(m.Prefix) > 0 {
		output += ":" + m.Prefix + " "
	}

	output += m.Command

	for _, param := range m.Params {
		output += " " + encodeParam(param)
	}

	return output
}

// ParseMessage parses a given IRC line (without the ending CR LF) and returns a
// a Message object.
func ParseMessage(line string) (*Message, error) {
	// Sanity check
	if len(line) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	message := new(Message)

	// Check if there is a prefix
	if line[0] == ':' {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 1 {
			return nil, io.ErrUnexpectedEOF
		}

		message.Prefix = parts[0][1:]
		line = parts[1]
	}

	// Split out the long argument
	halves := strings.SplitN(line, " :", 2)
	// Strip spaces
	halves[0] = strings.TrimSpace(halves[0])

	// Split up the command
	pieces := strings.Split(halves[0], " ")
	message.Command = pieces[0]
	message.Params = pieces[1:]

	// Append the long argument
	if len(halves) == 2 {
		message.Params = append(message.Params, halves[1])
	}

	return message, nil
}
