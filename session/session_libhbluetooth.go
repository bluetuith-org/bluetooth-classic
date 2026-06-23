//go:build !linux && libhbluetooth

package session

import (
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/internal/libhbluetooth"
)

// NewSession returns a platform-specific session handler.
func NewSession() bluetooth.Session {
	return &libhbluetooth.BluetoothLibrary{}
}
