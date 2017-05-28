package fmtp

import (
	"bytes"
	"io"
)

// inAgent receives incoming messages in a reader & unmarshal them, sending them as message when they are ready
// reading & unmarshalling is tightly coupled as TCP is a streaming protocol, so we can't use a pipeline infrastructure here.
func inAgent(in io.Reader, done chan struct{}, buffer int) (out chan *Message, errChan chan error) {
	// Create the return channels
	out = make(chan *Message, buffer)
	errChan = make(chan error)

	// Launch the goroutine
	go func(in io.Reader, done chan struct{}, out chan *Message, errChan chan error) {
		for {
			select {
			default:
				msg := &Message{}
				_, err := msg.ReadFrom(in)
				if err != nil {
					errChan <- err
				} else {
					out <- msg
				}
			case <-done:
				close(out)
				close(errChan)
				return
			}
		}
	}(in, done, out, errChan)

	// Return the channels
	return
}

// outAgent receives outgoing messages, marshals them and writes them to a writer
// it uses marshalAgent and writeAgent to make it pretty and efficient
func outAgent(w io.Writer, in chan *Message, done chan struct{}, buffer int) chan error {
	// Create the marshaller
	binChan, binErrChan := marshalAgent(in, done, buffer)

	// Create the writer
	wErrChan := writeAgent(w, binChan, done)

	// Merge the two error chan
	errChan := make(chan error)
	go func(bec, wec, ec chan error, done chan struct{}) {
		for {
			select {
			case err := <-bec:
				ec <- err
			case err := <-wec:
				ec <- err
			case <-done:
				return
			}
		}
	}(binErrChan, wErrChan, errChan, done)

	// Return the channels
	return errChan
}

// marshalAgent marshals messages
func marshalAgent(in chan *Message, done chan struct{}, buffer int) (out chan []byte, errChan chan error) {
	// Create the return channels
	out = make(chan []byte, buffer)
	errChan = make(chan error)

	// Launch the goroutine
	go func(in chan *Message, done chan struct{}, out chan []byte, errChan chan error) {
		for {
			select {

			case msg := <-in:
				buf := &bytes.Buffer{}
				_, err := msg.WriteTo(buf)
				if err != nil {
					errChan <- err
				} else {
					out <- buf.Bytes()
				}
			case <-done:
				close(out)
				close(errChan)
				return
			}
		}
	}(in, done, out, errChan)

	// Return the channels
	return
}

// writeAgent writes incoming []byte to a given io.Writer
func writeAgent(w io.Writer, in chan []byte, done chan struct{}) chan error {
	// Create the error return channel
	errChan := make(chan error)

	// Launch the goroutine
	go func(w io.Writer, in chan []byte, done chan struct{}, errChan chan error) {
		for {
			select {
			case b := <-in:
				_, err := w.Write(b)
				if err != nil {
					errChan <- err
				}
			case <-done:
				close(errChan)
				return
			}
		}
	}(w, in, done, errChan)

	// Return
	return errChan
}
