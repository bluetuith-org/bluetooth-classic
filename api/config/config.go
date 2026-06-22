package config

import (
	"time"
)

const (
	// DefaultAuthTimeout is the default timeout duration for authentication requests.
	DefaultAuthTimeout = 10 * time.Second
)

// Configuration describes a general configuration.
type Configuration struct {
	// SocketPath holds the user-defined path to the socket used to interface with the 'haraltd' daemon.
	SocketPath string

	// AuthTimeout holds the timeout for authentication requests.
	AuthTimeout time.Duration

	// LibraryPath holds the custom user-defined path for the 'libhbluetooth' library.
	LibraryPath string
}

// New returns a new configuration with the default authentication timeout.
func New() Configuration {
	return Configuration{
		AuthTimeout: DefaultAuthTimeout,
	}
}
