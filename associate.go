package fmtp

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// ErrAssociationTimeoutExceeded happens when the association reception timeout (tr) is exceeded
	// It is returned when associating. Once associated, such error will shutdown the association.
	ErrAssociationTimeoutExceeded = errors.New("association reception timeout exceeded")
)

// initAssociate establishes an FMTP association, without locking !
func (conn *Conn) initAssociate(ctx context.Context, recv <-chan *Message) error {
	// Debug
	logger := conn.client.logger.WithFields(logrus.Fields{
		"addr":      conn.RemoteAddr().String(),
		"remote ID": conn.RemoteID(),
	})
	logger.Debug("initAssociate called")

	logger.Debug("creating STARTUP request")
	// Create a STARTUP request
	msg, err := newSystemMessage(startup)
	if err != nil {
		return errors.Wrap(err, "Associate: error while creating system message")
	}

	logger.Debug("sending STARTUP request")
	// Send it
	err = conn.send(ctx, msg)
	if err != nil {
		return err
	}
	logger.Debug("send successful, waiting for response")

	// Wait for a STARTUP response
	var reply *Message
	select {
	case reply = <-recv:
	case <-ctx.Done():
		return ctx.Err()
	}
	logger.Debug("response retrieved")

	// Unmarshal it
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, reply.Body)
	if err != nil {
		return err
	}
	ss := &systemSig{}
	err = ss.UnmarshalBinary(buf.Bytes())
	if err != nil {
		return err
	}

	// Check if its startup
	if !ss.equals(startup) {
		return errors.New("system message not startup")
	}

	logger.Debug("connection initiated")
	return nil
}

// recvAssociate establishes an association requested by the peer, after the first STARTUP has been received.
func (conn *Conn) recvAssociate(ctx context.Context) error {
	conn.client.logger.Debug("recvAssociate called")
	conn.client.logger.Debug("creating STARTUP")
	// Create a STARTUP request
	msg, err := newSystemMessage(startup)
	if err != nil {
		return errors.Wrap(err, "Associate: error while creating system message")
	}
	conn.client.logger.Debug("STARTUP created")

	conn.client.logger.Debug("sending STARTUP")
	// Send it
	err = conn.send(ctx, msg)
	if err != nil {
		return err
	}
	conn.client.logger.Debug("STARTUP sent")

	// OK
	return nil
}

// deassociate is the actual action taken by an agent when deassociating
//
// what really happens is that a SHUTDOWN is sent
func (conn *Conn) deassociate(ctx context.Context) error {
	// Create a SHUTDOWN message
	msg, err := newSystemMessage(shutdown)
	if err != nil {
		return errors.Wrap(err, "deassociate: error while creating system message")
	}

	// Send it
	return conn.send(ctx, msg)
}
