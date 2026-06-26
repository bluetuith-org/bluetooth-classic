//go:build !linux && libhbluetooth

package lib

import (
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/ebitengine/purego"
)

func newCallback(fn any) uintptr {
	return purego.NewCallback(fn)
}

type eventCallbacks struct {
	adapterEventCb uintptr
	deviceEventCb  uintptr
	authEventCb    uintptr
	obexEventCb    uintptr
}

func newEventCallbacks() (*eventCallbacks, error) {
	e := &eventCallbacks{}

	return e, e.setup()
}

func (e *eventCallbacks) setup() error {
	e.adapterEventCb = newCallback(func(eventAction nativeEventAction, adapterData *adapterNative) uintptr {
		data := adapterData.toAdapterData()
		store := _libHandle.store

		switch nativeEventAction(eventAction).ToEventAction() {
		case bluetooth.EventActionAdded:
			store.AddAdapter(data)
			bluetooth.AdapterEvents().PublishAdded(data)

		case bluetooth.EventActionUpdated:
			if ev, err := store.UpdateAdapter(data.AdapterAddress, func(ad *bluetooth.AdapterData) error {
				return ad.Merge(&data)
			}); err == nil {
				bluetooth.AdapterEvents().PublishUpdated(ev)
			}

		case bluetooth.EventActionRemoved:
			store.RemoveAdapter(data.AdapterAddress)
			bluetooth.AdapterEvents().PublishRemoved(data.AdapterEventData)
		}

		return 0
	})

	e.deviceEventCb = newCallback(func(eventAction nativeEventAction, deviceData *deviceNative) uintptr {
		data := deviceData.ToDeviceData()
		store := _libHandle.store

		switch eventAction.ToEventAction() {
		case bluetooth.EventActionAdded:
			store.AddDevice(data)
			bluetooth.DeviceEvents().PublishAdded(data)

		case bluetooth.EventActionUpdated:
			if ev, err := store.UpdateDevice(data.DeviceAddress, func(dd *bluetooth.DeviceData) error {
				return dd.Merge(&data)
			}); err == nil {
				bluetooth.DeviceEvents().PublishUpdated(ev)
			}

		case bluetooth.EventActionRemoved:
			store.RemoveDevice(data.DeviceAddress)
			bluetooth.DeviceEvents().PublishRemoved(data.DeviceEventData)
		}

		return 0
	})

	e.authEventCb = newCallback(func(eventType authEventType, argRequest unsafe.Pointer) uintptr {
		handleAuthEvent(eventType, argRequest)
		return 0
	})

	e.obexEventCb = newCallback(func(obexEventType obexProfile, eventAction nativeEventAction, argObexData unsafe.Pointer) uintptr {
		handleObexEvent(obexEventType, eventAction, argObexData)
		return 0
	})

	return nil
}
