//go:build !linux

package commands

// Option describes an option to a command.
type Option string

// The various types of options.
const (
	SocketOption           Option = "--socket-path"
	AddressOption          Option = "--address"
	StateOption            Option = "--state"
	ProfileOption          Option = "--uuid"
	FileOption             Option = "--file"
	AuthenticationIdOption Option = "--authentication-id"
	ResponseOption         Option = "--response"
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
