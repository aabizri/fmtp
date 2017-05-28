package fmtp

import (
	"context"
	"net"

	"github.com/pkg/errors"
)

// establishTCPConn is a helper function to establish a TCP connection
func establishTCPConn(ctx context.Context, dialer *net.Dialer, address string) (*net.TCPConn, error) {
	// Establish TCP connection
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	// Assert it as a TCP conn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, errors.New("establishTCPConn: net.Conn isn't net.TCPConn")
	}

	// Set the connection to the appropriate options
	err = tcpConn.SetKeepAlive(false) //TMP: should be true
	if err != nil {
		return nil, errors.Wrap(err, "establishTCPConn: error while setting keep-alive")
	}

	return tcpConn, nil
}
