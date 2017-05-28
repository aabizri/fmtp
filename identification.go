package fmtp

import (
	"github.com/pkg/errors"
)

const (
	maxIDLen = 32
	minIDLen = 1
)

// An ID (Identification Value) is of maximum 32 Byte length
type ID string

// Check checks an ID validity
func (id *ID) Check() error {
	l := len(*id)
	switch {
	case l > 32:
		return errors.Errorf("ID.Check: ID too long (max is 32, this is %d)", l)
	case l == 0:
		return errors.New("ID.Check: Empty ID")
	}
	return nil
}
