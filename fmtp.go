/*Package fmtp provides Flight Message Transfer Protocol (FMTP) support.
It currently supports v2.0

Note that the FMTP protocol is a layer 5,6,7 protocol in the OSI stack.
*/
package fmtp

type Typ uint8

// The following constants define the type of the message being carried
const (
	_ Typ = iota
	Operational
	Operator
	identification
	system
	// should we include status messages ? typ=5 ?
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
