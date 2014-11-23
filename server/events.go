package server

import "github.com/sorcix/irc"

// Server events

// SRegisterClient is used to inform the server that a client has sent all
// registration information and is ready to be registered. A boolean is sent on
// the reply channel stating whether or not this registration was successful.
type SRegisterClient struct {
	Client *Client
	Reply  chan bool
}

// SReregisterClient is used to inform the server that a client has changed
// their nickname and needs to be reregistered.
type SReregisterClient struct {
	NewNick string
	Client  *Client
	Reply   chan bool
}

// SDeregisterClient is used to inform the server that a client has disconnected
// and registration information needs to be discarded.
type SDeregisterClient struct {
	Client *Client
	Reply  chan struct{}
}

// Client events

// CInitialize is used to inform the Client that it needs to initialize. These
// include pre-registration tasks such as looking up the hostname and checking
// identd.
type CInitialize struct{}

// CMessage is used to inform the Client of an incoming message that has been
// parsed and is ready to be acted upon.
type CMessage struct {
	Message *irc.Message
}

// CClose is used to inform the Client that it must immediately close the
// connection.
// Reason is used in QUIT messages and sent to the connection, if possible.
type CClose struct {
	Reason string
}
