package fmtp

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"bufio"

	"github.com/pkg/errors"
)

// A Message is an FMTP message
type Message struct {
	header *header
	Body   io.Reader
}

// bodyLen returns the size of the body if it can find it, it returns 0, false when it isn't defined
func (msg *Message) bodyLen() (uint16, bool) {
	// If we have no header or message is nil, we return 0, false
	if msg == nil || msg.header == nil {
		return 0, false
	}

	// If the len is indicated in the header, use it
	if bLen := msg.header.bodyLen(); bLen != 0 {
		return uint16(bLen), true
	}

	// Establish the interfaces
	// Not Size() as Size returns the size of the underlying data, but not how much you'll read.
	type lener interface {
		Len() int
	}
	type byteser interface {
		Bytes() int
	}

	// Switch over the interfaces
	switch r := msg.Body.(type) {
	case lener:
		return uint16(r.Len()), true
	case byteser:
		return uint16(r.Bytes()), true
	}

	// If we didn't find anything, return 0, false
	return 0, false
}

// WriteTo writes a Message to the given io.Writer.
// This consumes the Message Body.
func (msg *Message) WriteTo(w io.Writer) (int64, error) {
	// Check if message is valid
	if msg.header == nil {
		return 0, errors.New("WriteTo: cannot write message as header is nil")
	}

	// Read the body into a []byte
	bbin, err := ioutil.ReadAll(msg.Body)
	if err != nil {
		return 0, err
	}

	// Set the correct body length in the header
	msg.header.setBodyLen(uint16(len(bbin)))

	// Marshal it
	hbin, err := msg.header.MarshalBinary()
	if err != nil {
		return 0, err
	}

	// Write using a bufio.Writer
	wbuf := bufio.NewWriterSize(w, len(bbin)+len(hbin))
	nb1, err := wbuf.Write(hbin)
	if err != nil {
		return 0, err
	}
	nb2, err := wbuf.Write(bbin)
	if err != nil {
		return 0, err
	}
	total := int64(nb1 + nb2)

	// Flush !
	err = wbuf.Flush()
	if err != nil {
		return total, err
	}

	return total, nil
}

// ReadFrom creates a m.Message from an io.Reader.
func (msg *Message) ReadFrom(r io.Reader) (int64, error) {
	// First we decode the header
	h := &header{}
	b := make([]byte, headerLen)

	// Read the expected length
	n1, err := r.Read(b)
	if err != nil {
		return int64(n1), err
	}

	// Unmarshal it
	err = h.UnmarshalBinary(b)
	if err != nil {
		return int64(n1), err
	}
	msg.header = h

	// Now, given the header-indicated size we create a buffer of that size
	bodyLen := h.bodyLen()
	content := make([]byte, bodyLen)
	n2, err := io.ReadFull(r, content)
	total := int64(n1 + n2)
	if err != nil {
		return total, err
	} else if n2 < bodyLen {
		return total, errors.Errorf("ReadFrom: message body is smaller than expected (%d<%d)", n2, bodyLen)
	}

	// And we create a bytes.Reader from it
	body := bytes.NewReader(content)
	msg.Body = body

	return total, nil
}

// Typ returns the message's type
func (msg *Message) Typ() Typ {
	if msg == nil || msg.header == nil {
		panic("cannot extract type from nil message")
	}
	return msg.header.typ
}

// NewMessage returns a message of either Operational or Operator type
func NewMessage(typ Typ, r io.Reader) (*Message, error) {
	return &Message{
		header: newHeader(typ),
		Body:   r,
	}, nil
}

// NewOperationalMessage returns a message of Operational type
func NewOperationalMessage(r io.Reader) (*Message, error) {
	return NewMessage(Operational, r)
}

// NewOperatorMessage returns a message of Operator type
func NewOperatorMessage(r io.Reader) (*Message, error) {
	// TODO: Embed it in a reader checking for ASCII-only text
	return NewMessage(Operator, r)
}

// NewOperatorMessageString returns a message of Operator type built from the given string
func NewOperatorMessageString(txt string) (*Message, error) {
	r := strings.NewReader(txt)
	msg, err := NewMessage(Operator, r)
	if err != nil {
		return msg, err
	}
	msg.header.setBodyLen(uint16(len(txt)))
	return msg, nil
}

// newIDRequestMessage returns an identification request message
func newIDRequestMessage(sender ID, receiver ID) (*Message, error) {
	// Create the payload
	idr, err := newidRequest(sender, receiver)
	if err != nil {
		return nil, errors.Wrap(err, "newIDRequestMessage: error while creating new IDRequest message")
	}

	// Marshal it
	bin, err := idr.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "newIDRequestMessage: error when marshalling id request")
	}

	return NewMessage(identification, bytes.NewReader(bin))
}

// newIDResponseMessage returns an identification response message
func newIDResponseMessage(accept bool) (*Message, error) {
	// Create the payload
	idr := newidResponse(accept)

	// Marshal it
	bin, err := idr.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "newIDResponseMessage: error when recovering reader for id response")
	}

	return NewMessage(identification, bytes.NewReader(bin))
}

// newSystemMessage returns a system message
func newSystemMessage(ss *systemSig) (*Message, error) {
	return NewMessage(system, bytes.NewReader(ss[:]))
}
