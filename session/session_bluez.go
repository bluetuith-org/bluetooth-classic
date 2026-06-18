//go:build linux

package session

import (
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/bluez"
)

// NewSession returns a Bluez-specific session handler.
func NewSession() bluetooth.Session {
	return &bluez.BluezSession{}
}
