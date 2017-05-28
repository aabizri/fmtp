package fmtp

import (
	"context"
	"log"
	"net"
	"time"
)

// A Handler receives and processes an FMTP message.
type Handler interface {
	ServeFMTP(conn *Conn, msg *Message)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as FMTP handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(conn *Conn, msg *Message)

// ServeFMTP satisfies the Handler interface
func (hf HandlerFunc) ServeFMTP(conn *Conn, msg *Message) {
	hf(conn, msg)
}

// A Server defines parameters for running an FMTP server.
type Server struct {
	// TCP address to listen on
	Addr string

	// Handler is the handler for new connections.
	Handler Handler

	// Timeouts
	Ti time.Duration
	Ts time.Duration
	Tr time.Duration

	// AcceptTCP is called when a new TCP connection is inbound
	// If AcceptTCP is nil, every incoming connections are accepted
	AcceptTCP func(remoteAddr net.Addr) bool

	// NotifyConn is called when a connection was successfuly established
	NotifyConn func(remoteAddr net.Addr, remoteID ID)

	// Done
	done chan struct{}

	// Client
	c *Client
}

// NewServer creates a server for use
func (c *Client) NewServer(addr string, h Handler) *Server {
	return &Server{c: c, Addr: addr, Handler: h}
}

// ListenAndServe listens to an IP Address, and handles functions
func (srv *Server) ListenAndServe() error {
	// Resolve the address
	laddr, err := net.ResolveTCPAddr("tcp", srv.Addr)
	if err != nil {
		return err
	}

	// First, bind to the given address
	l, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}

	// Now launch serve on it
	defer l.Close()
	return srv.Serve(l)
}

// logf logs server errors
func (srv *Server) logf(format string, params ...interface{}) {
	log.Printf(format, params...)
}

// Serve serves incoming connections on a net.Listener
func (srv *Server) Serve(l *net.TCPListener) error {
	var tempDelay time.Duration
	for {
		// We accept the next connection
		rw, e := l.AcceptTCP() //rw, e

		// Check for errors
		if e != nil {
			select {
			case <-srv.done:
				return nil
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.c.logger.Error("Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0

		// We ask for agreement for the remote IP using AcceptRemoteIP.
		// If there is no such function, it's like a whitelist.
		if srv.AcceptTCP != nil {
			ok := srv.AcceptTCP(rw.RemoteAddr())
			if !ok {
				// TODO: Do something if there's an error
				err := rw.Close()
				_ = err
			}
		}

		// We have a new TCP conn, so we register it
		go srv.registerTCPConn(rw)
	}
}

// registerTCPConn registers a new TCP connection, accepting the connection
func (srv *Server) registerTCPConn(tcp *net.TCPConn) {
	// Create a new connection
	conn := srv.c.NewConn(srv.Handler)

	// Set the current transport as the underlying one
	conn.SetUnderlying(tcp)

	// Launch process of incoming connection
	err := conn.recv(context.Background())
	if err != nil {
		tcp.Write([]byte("ERROR: ILLEGAL\n"))
		tcp.Close()
	}

	// Notify that a connection has been made
	if srv.NotifyConn != nil {
		go srv.NotifyConn(conn.RemoteAddr(), conn.RemoteID())
	}
}

// Shutdown stops the server gracefully
func (srv *Server) Shutdown(ctx context.Context) error {
	// Closes all open listeners (stopping new connections from forming)
	// Send the Shutdown to all idle Connections
	// Waits for every Connection, idle or associated, to close which can be forever
	return nil
}

// Close stops the server
func (srv *Server) Close() error {
	// Closes all active listeners and all connections, associated or not
	return nil
}
