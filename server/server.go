// Package server contains the server functionality.
package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	"code.google.com/p/go-uuid/uuid"

	"github.com/nightexcessive/excessiveircd/config"
)

// ListenPort represents a port to be listened on.
// IP specifies the local IP address to listen on.
// Port specifies the port number to listen on.
// TLS specifies the TLS configuration to use. If nil, the port listens for
// cleartext traffic.
type ListenPort struct {
	IP   net.IP
	Port uint16

	TLS *tls.Config
}

// Server represents a local server.
type Server struct {
	ID      uuid.UUID
	Name    string
	Network string

	Logger *log.Logger

	Events chan interface{}

	Clients map[string]*Client

	Listeners []net.Listener
}

// FriendlyName is a convenience function to return the server's display name.
func (s *Server) FriendlyName() string {
	if len(s.Name) > 0 {
		return s.Name
	}

	return s.ID.String()
}

func (s *Server) eventLoop() {
	for event := range s.Events {
		s.Logger.Printf("Event: %T", event)
		switch ev := event.(type) {
		case *SRegisterClient:
			if _, ok := s.Clients[ev.Client.Info.Name]; ok {
				ev.Reply <- false
				continue
			}
			s.Clients[ev.Client.Info.Name] = ev.Client
			ev.Reply <- true
		default:
			s.Logger.Printf("Unexpected event of type %T: %#v", ev, ev)
		}
	}
}

// Start starts the server's event loop and all of its listeners.
func (s *Server) Start() error {
	s.Events = make(chan interface{})
	defer close(s.Events)

	s.Clients = make(map[string]*Client)

	go s.eventLoop()

	var listeners []*ListenPort
	if err := config.Get("ports", &listeners); err == config.ErrDoesNotExist {
		listeners = []*ListenPort{
			{
				IP:   net.IPv4(0, 0, 0, 0),
				Port: 6667,
			},
		}
	} else if err != nil {
		return err
	}

	if err := config.Get("id", &s.ID); err == config.ErrDoesNotExist {
		s.ID = uuid.NewRandom()
		if err := config.Set("id", s.ID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	s.Logger = log.New(os.Stderr, fmt.Sprintf("Server(%s) ", s.ID), log.LstdFlags)

	s.startListeners(listeners)

	return nil
}

func (s *Server) close(reason string) error {
	s.Logger.Printf("Shutting down: %s", reason)
	for _, listener := range s.Listeners {
		if err := listener.Close(); err != nil {
			s.Logger.Printf("Error closing listener on %s: %s", listener.Addr(), err)
		}
	}

	// This should be safe because the event should never be modified.
	closeEvent := &CClose{"Server shutting down"}
	if len(reason) > 0 {
		closeEvent.Reason = fmt.Sprintf("Server shutting down: %s", reason)
	}
	wg := new(sync.WaitGroup)
	wg.Add(len(s.Clients))
	for _, client := range s.Clients {
		go func(client *Client) {
			defer wg.Done()
			client.Events <- closeEvent
		}(client)
	}
	wg.Wait()

	close(s.Events)

	return nil
}

func (s *Server) listen(listenSpec *ListenPort, wg *sync.WaitGroup) {
	defer wg.Done()

	listenAddr := net.JoinHostPort(listenSpec.IP.String(), strconv.FormatInt(int64(listenSpec.Port), 10))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		s.Logger.Printf("Failed to listen on %s: %s", listenAddr, err)
		return
	}
	defer listener.Close()
	s.Logger.Printf("Listening on %s...", listenAddr)

	for {
		c, err := listener.Accept()
		if err != nil {
			s.Logger.Printf("Error accepting connection on %s: %s", listenAddr, err)
			continue
		}

		s.Logger.Printf("New connection to %s from %s", listenAddr, c.RemoteAddr())
		go NewClient(c, s)
	}
}

func (s *Server) startListeners(listeners []*ListenPort) {
	wg := new(sync.WaitGroup)
	wg.Add(len(listeners))

	for _, listener := range listeners {
		go s.listen(listener, wg)
	}

	wg.Wait()
}

// Start starts the primary server. This blocks until the server stops. This
// should only be called once.
func Start() {
	server := new(Server)
	if err := server.Start(); err != nil {
		panic(err)
	}
}
