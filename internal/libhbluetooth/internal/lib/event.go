//go:build !linux && libhbluetooth

package lib

import "github.com/bluetuith-org/bluetooth-classic/api/bluetooth"

type nativeEventAction uint32

const (
	eventActionNone nativeEventAction = iota
	eventActionAdded
	eventActionUpdated
	eventActionRemoved
)

func (n nativeEventAction) ToEventAction() bluetooth.EventAction {
	switch n {
	case eventActionNone:
		return bluetooth.EventActionNone

	case eventActionAdded:
		return bluetooth.EventActionAdded

	case eventActionUpdated:
		return bluetooth.EventActionUpdated

	case eventActionRemoved:
		return bluetooth.EventActionRemoved
	}

	return ""
}
