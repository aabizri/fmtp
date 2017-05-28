// the agent is a connection's supervisor

package fmtp

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

// agent is the manager of a connection
// 	- In its unassociated form, it listens to STARTUP commands
// 	- When associated, it maintains the association and handles incoming/outgoing messages
//
// Problem: function too big, one way to fix it would be to split it into three:
// 	- Agent (manages processes like association/deassociation/disconnections, and higher-level send/receive operations)
// 	- Outgoing (send messages)
// 	- Incoming (received messages)
func (conn *Conn) agent() {
	var (
		// whether the connection is associated
		associated bool
		// Create the ts timer for heartbeats
		ts = &time.Timer{}
	)

	// Create a watchdog context, which we will be able to cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch a listener goroutine
	msgChan := make(chan *Message)
	errChan := make(chan error)
	defer func() {
		close(msgChan)
		close(errChan)
	}()
	go func(ctx context.Context, mc chan *Message, ec chan error) {
		for {
			msg, err := conn.receive(ctx)
			switch err {
			case io.EOF: // do nothing
			case context.Canceled, context.DeadlineExceeded:
				return
			case nil:
				if ctx.Err() != nil {
					return
				}
				mc <- msg
			default:
				ec <- err
				conn.Close()
			}
		}
	}(ctx, msgChan, errChan)

	// Event loop, checking for arrival & new orders
	for {
		select {
		// If we received a message, we handle it
		case msg := <-msgChan:
			switch msg.header.typ {
			// If it is a system message, we handle it
			case system:
				// Write to buffer
				buf := &bytes.Buffer{}
				_, err := io.Copy(buf, msg.Body)
				if err != nil {
					fmt.Printf("error while copying message body: %v\n", err)
				}

				// Unmarshal
				ss := &systemSig{}
				err = ss.UnmarshalBinary(buf.Bytes())
				if err != nil {
					fmt.Printf("error while unmarshalling system message: %v\n", err)
				}

				// Compare
				switch {
				case ss.equals(startup):
					err := conn.recvAssociate(ctx)
					fmt.Println(err)
					associated = true
					ts = time.NewTimer(conn.Tr)
					_ = err
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

		// In case we get got an order, process it
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

		// If we get a done signal, we close immediately, but not before emptying the orders channel
		case <-conn.done:
			for {
				select {
				case o := <-conn.orders:
					o.done <- errors.New("connection is closing")
				default:
					close(conn.orders)
					return
				}
			}
		}
	}
}

// handleErr dispatches an error in the handling to the user
func (conn *Conn) handleErr(err error) {}
