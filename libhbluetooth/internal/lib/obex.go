//go:build !linux && libhbluetooth

package lib

import "unsafe"

type obexProfile uint32

const (
	none obexProfile = iota
	profileObjectPush
)

type obexEvent[T any] struct {
	data *T
}

func handleObexEvent(obexProfile obexProfile, eventAction nativeEventAction, eventData unsafe.Pointer) {
	action := eventAction.ToEventAction()

	switch obexProfile {
	case profileObjectPush:
		oppData := resolveDataPointer[obexEvent[oppTransferData]](eventData)
		handleOppEvent(action, oppData.data)

	default:
	}
}
