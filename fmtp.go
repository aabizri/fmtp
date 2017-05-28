// Package fmtp provides Flight Message Transfer Protocol (FMTP) support.
// It currently supports v2.0
package fmtp

type Typ uint8

// The following constants define the type of the message being carried
const (
	_ Typ = iota
	Operational
	Operator
	identification
	system
)

func (t Typ) String() string {
	switch t {
	case Operational:
		return "Operational"
	case Operator:
		return "Operator"
	case identification:
		return "Identification"
	case system:
		return "System"
	default:
		return "Unknown Typ"
	}
}
