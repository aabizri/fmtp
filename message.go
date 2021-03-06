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
	Body   io.ReadCloser
}

// readerLen returns the size of a reader if it can find it
func readerLen(r io.Reader) (int, bool) {
	// Establish the interfaces
	// Not Size() as Size returns the size of the underlying data, but not how much you'll read.
	type lener interface {
		Len() int
	}
	type byteser interface {
		Bytes() int
	}

	// Switch over the interfaces
	switch r := r.(type) {
	case lener:
		return r.Len(), true
	case byteser:
		return r.Bytes(), true
	}

	return 0, false
}

// bodyLen returns the size of the body if it can find it, it returns 0, false when it isn't defined
func (msg *Message) bodyLen() (int, bool) {
	// If we have no header or message is nil, we return 0, false
	if msg == nil || msg.header == nil {
		return 0, false
	}

	// If the len is indicated in the header, use it
	if bLen := msg.header.bodyLen(); bLen != 0 {
		return bLen, true
	}

	// ReaderLen is the len of the reader
	rlen, found := readerLen(msg.Body)

	// If we didn't find anything, return 0, false
	return rlen, found
}

// WriteTo writes a Message to the given io.Writer.
// This consumes the Message Body.
func (msg *Message) WriteTo(w io.Writer) (int64, error) {
	// Check if message is valid
	if msg.header == nil {
		return 0, errors.New("WriteTo: cannot write message as header is nil")
	}

	// Read the body into a []byte
	r := io.LimitReader(msg.Body, MaxBodyLen+1)
	defer msg.Body.Close()
	bbin, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	} else if len(bbin) > MaxBodyLen {
		return 0, errors.New("WriteTo: cannot write message as body is larger than MaxBodyLen")
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
	body := ioutil.NopCloser(bytes.NewReader(content))
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
// See MaxBodyLen for the maximum size of a message's body
func NewMessage(typ Typ, r io.Reader) (*Message, error) {
	// Create the header
	header := newHeader(typ)

	// Advance warning if we can extract the length of the reader
	blen, found := readerLen(r)
	if found && blen > MaxBodyLen {
		return nil, errors.Errorf("message body length bigger than maximum (%d > %d)", blen, MaxBodyLen)
	} else if found {
		header.setBodyLen(uint16(blen))
	}

	// If the given reader is a closer, we use it directly
	var rc io.ReadCloser
	if trc, ok := r.(io.ReadCloser); ok {
		rc = trc
	} else {
		rc = ioutil.NopCloser(r)
	}

	// Done
	return &Message{
		header: header,
		Body:   rc,
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
