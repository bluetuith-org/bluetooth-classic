//go:build !linux && haraltd

package session

import (
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/haraltd"
)

// NewSession returns a platform-specific session handler.
func NewSession() bluetooth.Session {
	return &haraltd.HaraltdSession{}
}
