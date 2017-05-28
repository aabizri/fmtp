package fmtp

import (
	"github.com/pkg/errors"
)

var (
	startup   = &systemSig{'0', '1'}
	shutdown  = &systemSig{'0', '0'}
	heartbeat = &systemSig{'0', '3'}
)

type systemSig [2]byte

func (ss *systemSig) MarshalBinary() ([]byte, error) {
	return ss[:], nil
}

func (ss *systemSig) UnmarshalBinary(b []byte) error {
	if len(b) != 2 {
		return errors.Errorf("UnmarshalBinary: expected byte slice length %d, got %d", 2, len(b))
	}
	ss[0] = b[0]
	ss[1] = b[1]
	return nil
}

func (ss *systemSig) equals(other *systemSig) bool {
	return ss[0] == other[0] && ss[1] == other[1]
}
