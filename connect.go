// connect.go manages the process of connecting
// Connection establishment overview:
// 	- First a TCP transport is established with the remote FMTP entity
// 	- The responder starts a local timer Ti.
// 	- The initator sends an identification message, and starts a local timer Ti.
// 	- The responder validates the received Identification message, replies by sending an identification message back to the initiator
//	and resetting Ti
// 	- The initiator validates the received identification message, sends an Identification ACCEPT to the responder, and stops Ti
// 	- The responder receives the ACCEPT and stops Ti
// 	- Both endpoints confirm that the connection is established

package fmtp

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
)

// sendIDRequestMessage sends an IDRequestMessage
func (conn *Conn) sendIDRequestMessage(ctx context.Context, local ID, remote ID) error {
	// Create an identification message
	msg, err := newIDRequestMessage(local, remote)
	if err != nil {
		return err
	}

	// Send the identification message
	return conn.send(ctx, msg)
}

// recvIDRequestMessage receives an IDRequestMessage and extracts it from the message
func (conn *Conn) recvIDRequestMessage(ctx context.Context) (*idRequest, error) {
	// Receive the reply
	msg, err := conn.receive(ctx)
	if err != nil {
		return nil, err
	}

	// If it isn't an ID message, it's an error
	if msg.header != nil && msg.header.typ != identification {
		return nil, errors.New("received message isn't of correct typ")
	}

	// Unmarshal the reply body
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, msg.Body)
	if err != nil {
		return nil, err
	}
	idr := &idRequest{}
	idr.UnmarshalBinary(buf.Bytes())

	// Return it
	return idr, nil
}

// sendIDResponseMessage sends an Identification Response message.
func (conn *Conn) sendIDResponseMessage(ctx context.Context, ok bool) error {
	// Create an identification message
	msg, err := newIDResponseMessage(ok)
	if err != nil {
		return errors.Wrap(err, "Connect: error while creating identification message")
	}

	// Send the identification message
	return conn.send(ctx, msg)
}

// recvIDResponseMessage receives an an Identification Response message and unmarshals it
func (conn *Conn) recvIDResponseMessage(ctx context.Context) (*idResponse, error) {
	// Receive the reply
	msg, err := conn.receive(ctx)
	if err != nil {
		return nil, err
	}

	// Copy the message body to a buffer
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, msg.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the body
	idresp := &idResponse{}
	err = idresp.UnmarshalBinary(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Return it
	return idresp, nil
}

// recv receives a connection request from an outside party
func (conn *Conn) recv(ctx context.Context) error {
	// We create a local context following the ti timer
	tiCtx, cancel := context.WithTimeout(ctx, conn.Ti)
	defer cancel()

	// Receive an ID Request, using the tiCtx
	idr, err := conn.recvIDRequestMessage(tiCtx)
	if tiCtx.Err() != nil { // If the cancel comes from tiCtx, we do not return a "context canceled" but the correct error
		return ErrConnectionDeadlineExceeded
	} else if err != nil {
		return err
	}

	// We note the remote ID in the connection
	switch {
	// If we have an acceptRemote function, then we use it !
	case conn.acceptRemote != nil:
		// If we don't accept it, send a reject message
		if !conn.acceptRemote(idr.Sender) {
			conn.sendIDResponseMessage(ctx, false)
			return ErrConnectionRejectedByLocal
		}
		fallthrough // Now that the check is done, we expect the same particular behaviour as a wildcard

	// If we have no acceptRemote function then its a wildcard
	case conn.acceptRemote == nil:
		conn.remote = idr.Sender

	}

	// We send an ID request message using the normal context
	err = conn.sendIDRequestMessage(ctx, conn.local, idr.Sender)
	if err != nil {
		return err
	}

	// We reset our tiCtx
	tiCtx, cancel = context.WithTimeout(ctx, conn.Ti)
	defer cancel()

	// We await a positive response
	idresp, err := conn.recvIDResponseMessage(tiCtx)
	if tiCtx.Err() != nil { // If the cancel comes from tiCtx, we do not return a "context canceled" but the correct error
		return ErrConnectionDeadlineExceeded
	} else if err != nil {
		return err
	}

	// If the response was negative, we signal it
	if !idresp.OK {
		return ErrConnectionRejectedByRemote
	}

	// launch the agent
	go conn.agent()

	return nil
}
