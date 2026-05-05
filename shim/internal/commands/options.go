//go:build !linux

package commands

// RPCAgent describes the type of authentication agent.
type RPCAgent string

// The different types of authentication agents.
const (
	PairingAgent RPCAgent = "pairing"
	ObexAgent    RPCAgent = "obex"
)

// String returns the string representation of the agent.
func (o RPCAgent) String() string {
	return string(o)
}

// Option describes an option to a command.
type Option string

// The various types of options.
const (
	SocketOption           Option = "--socket-path"
	AddressOption          Option = "--address"
	StateOption            Option = "--state"
	ProfileOption          Option = "--uuid"
	FileOption             Option = "--file"
	AuthenticationIDOption Option = "--authentication-id"
	ResponseOption         Option = "--response"
	AgentOption            Option = "--agent-type"
)

// String returns a string representation of the option.
func (a Option) String() string {
	return string(a)
}

// StateOptionValue returns the appropriate value to the 'StateOption'
// according to how the 'enable' parameter is set.
func StateOptionValue(enable bool) string {
	if !enable {
		return "off"
	}

	return "on"
}
