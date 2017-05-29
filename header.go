package fmtp

import (
	"fmt"

	"encoding/binary"

	"github.com/pkg/errors"
)

const (
	// values indicated in the specification
	version2  = 2
	reserved2 = 0

	// length of a header field
	headerLen = 5

	// maxLength that can be indicated
	maxLength = 65535

	// MaxBodyLen is the maximum body len
	MaxBodyLen = maxLength - headerLen
)

// header is a FMTP's message Header field
type header struct {
	// version indicated the header version.
	version uint8

	// reserved field is an internal value
	reserved uint8

	// length indicates the combined length in bytes of the Header and Body
	length uint16

	// typ indicates the message type that is being transferred
	typ Typ
}

func (h *header) Check() error {
	if h == nil {
		return nil
	}
	if h.length < headerLen {
		return errors.New("header.Check(): error, indicated length cannot be smaller than nominal header length")
	}
	return nil
}

// String prints the header
func (h *header) String() string {
	return fmt.Sprintf("Version:\t%d\nReserved:\t%d\nLength:\t%d bytes\n\tTyp:\t\t%d\n", h.version, h.reserved, h.length, h.typ)
}

// MarshalBinary marshals a header into binary form
func (h *header) MarshalBinary() ([]byte, error) {
	// Check
	err := h.Check()
	if err != nil {
		return nil, err
	}

	// Get the length in binary
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, h.length)

	// Now create the byte slice
	out := []byte{
		byte(h.version),
		byte(h.reserved),
		byte(lenBuf[0]),
		byte(lenBuf[1]),
		byte(h.typ),
	}
	return out, nil
}

func (h *header) UnmarshalBinary(b []byte) error {
	if len(b) != headerLen {
		return errors.Errorf("UnmarshalBinary: expected %d bytes, got %d", headerLen, len(b))
	}

	// Extract length
	length := binary.BigEndian.Uint16(b[2:4])
	if length > maxLength {
		return errors.New("UnmarshalBinary: indicated length larger than max length")
	} else if length < headerLen {
		return errors.New("UnmarshalBinary: indicated length smaller than nominal header length")
	}

	// Assign
	h.version = b[0]
	h.reserved = b[1]
	h.length = uint16(length)
	h.typ = Typ(b[4])

	return nil
}

// newHeader creates a new header in version 2.0
func newHeader(typ Typ) *header {
	return &header{
		version:  version2,
		reserved: reserved2,
		typ:      typ,
	}
}

// setLength sets the header length
func (h *header) setBodyLen(bodyLen uint16) {
	h.length = headerLen + bodyLen
}

// bodyLen returns the body length
// if no body is here
func (h *header) bodyLen() int {
	if h.length == 0 {
		return 0
	}
	return int(h.length) - headerLen
}
