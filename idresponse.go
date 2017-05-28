package fmtp

import (
	"github.com/pkg/errors"
)

const (
	accept     = "ACCEPT"
	reject     = "REJECT"
	keywordLen = 6
)

// An idResponse (Identification Message where the messages are being sent to respond to a validation request)
type idResponse struct {
	OK bool
}

// newidResponse returns an ID response
func newidResponse(accept bool) *idResponse {
	return &idResponse{accept}
}

// MarshalBinary marshals the idResponse
func (idr *idResponse) MarshalBinary() ([]byte, error) {
	switch idr.OK {
	case true:
		return []byte(accept), nil
	case false:
		return []byte(reject), nil
	}
	panic("unreachable code")
}

// UnmarshalBinary unmarshals the idResponse
func (idr *idResponse) UnmarshalBinary(in []byte) error {
	if idr == nil {
		return errors.New("idr is nil")
	}
	if len(in) != keywordLen {
		return errors.New("message not to correct size")
	}

	var val bool
	switch string(in) {
	case accept:
		val = true
	case reject:
		val = false
	default:
		return errors.New("message is neither ACCEPT nor REJECT")
	}

	idr.OK = val
	return nil
}
