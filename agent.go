package fmtp

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
)

// a command is what can be asked of the agent
type command uint8

const (
	// associate and deassociate commands
	associateCmd = iota
	deassociateCmd

	// disconnect disconnects
	disconnectCmd

	// send sends a message
	sendCmd
)

// an order is what's given to the agent to execute commands/send messages
type order struct {
	command command
	ctx     context.Context
	done    chan error
	msg     *Message
}

// order the agent to execute some command
// this is synchroneous
func (conn *Conn) order(ctx context.Context, command command) error {
	done := make(chan error)
	go func() {
		conn.orders <- order{
			command: command,
			ctx:     ctx,
			done:    done,
		}
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func handleSys(msg *Message) (ss *systemSig, err error) {
	// Write to buffer
	b, err := ioutil.ReadAll(msg.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal
	ss = &systemSig{}
	err = ss.UnmarshalBinary(b)
	return ss, err
}

// agent is the manager of a connection, it should implement a state machine with 6 states.
// the states are
// 	- idle (awaiting further instructions)
// 	- connPending (connection pending)
// 	- idPending (waiting for ID)
// 	- ready (connection established)
// 	- assPending (sent started, waiting for response)
// 	- dataReady (association established, ready to send data)
//
// TODO: Create a state machine with state functions
func (conn *Conn) agent() {
	var (
		// whether the connection is associated
		associated bool
		// Create the ts timer for heartbeats
		ts = &time.Timer{}
	)

	// Create the global context
	ctx := context.Background()

	// Launch the listener
	inDone := make(chan struct{})
	msgChan, errChan := inAgent(conn.tcp, inDone, 3)

	// Event loop, checking for arrival & new orders
	for {
		select {
		// If we received a message, we handle it
		case msg := <-msgChan:
			switch msg.header.typ {
			// If it is a system message, we handle it
			case system:
				// Unmarshal
				ss, err := handleSys(msg)
				_ = err

				// Compare
				switch {
				case ss.equals(startup):
					err := conn.recvAssociate(ctx)
					conn.handleErr(err)
					associated = true
					ts = time.NewTimer(conn.Tr)
				case ss.equals(heartbeat):
					// do something
				case ss.equals(shutdown):
					associated = false
					ts.Stop()
				}
			// If it is intended for the user, we pass it on
			case Operator, Operational:
				if !associated {
					conn.Close()
					return
				}
				if conn.Handler != nil {
					conn.Handler.ServeFMTP(conn, msg)
				}
			}

		// If we received an error, we evaluate it
		case err := <-errChan:
			conn.client.logger.Errorf("error in reception: %v", err)
			conn.handleErr(err)
			conn.Close()
			return

		// In case we get got an order, we process it
		case o := <-conn.orders:
			conn.client.logger.Debug("received new order")
			switch o.command {
			case disconnectCmd:
				err := conn.disconnect(o.ctx)
				o.done <- err
				return // By returning we call cancel()
			case associateCmd:
				err := conn.initAssociate(o.ctx, msgChan)
				if err == nil {
					associated = true
					ts = time.NewTimer(conn.Tr)
				}
				o.done <- err
			case deassociateCmd:
				err := conn.deassociate(o.ctx)
				if err == nil {
					associated = false
				}
				o.done <- err
			case sendCmd:
				// If we're not associated, we do so
				if !associated {
					err := conn.initAssociate(o.ctx, msgChan)
					if err == nil {
						associated = true
						ts = time.NewTimer(conn.Tr)
					}
				}
				// We send
				err := conn.send(o.ctx, o.msg)
				// We send the result back
				o.done <- err
				// We reset ts
				ts.Reset(conn.Ts)
			}

		// In case it's time to do a heartbeat, do it
		case <-ts.C:
			// If not associated, that's illegal
			if !associated {
				panic("HEARTBEAT TIMER ACTIVE EVEN THOUGH WE'RE NOT ASSOCIATED")
			}

			// Create a HEARTBEAT request
			msg, err := newSystemMessage(heartbeat)
			if err != nil {
				fmt.Println(errors.Wrap(err, "Associate: error while creating system message"))
				break
			}

			// Send it
			err = conn.send(ctx, msg)
			if err != nil {
				fmt.Println(err)
				break
			}

			// Reset timer
			ts.Reset(conn.Ts)

		// If we get a done signal, we close, but not before emptying the orders channel
		case <-conn.done:
			for {
				select {
				case o := <-conn.orders:
					o.done <- errors.New("connection is closing")
				default:
					inDone <- struct{}{}
					close(conn.orders)
					return
				}
			}
		}
	}
}

// handleErr dispatches an error in the handling to the user
func (conn *Conn) handleErr(err error) {}
