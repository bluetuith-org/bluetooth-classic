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

	// EnableObexServices holds a user-defined value that specifies whether to enable
	// OBEX related features. This option exists so that these services aren't unneccesarily
	// setup on every session creation.
	EnableObexServices bool
}

// New returns a new configuration with the default authentication timeout.
func New() Configuration {
	return Configuration{
		AuthTimeout: DefaultAuthTimeout,
	}
}
