//go:build !linux && libhbluetooth

package lib

import (
	"errors"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	ffi "github.com/bluetuith-org/libffi-go"
)

type callback struct {
	cl *ffi.Closure
	cb unsafe.Pointer
}

func newCallback(ffiCb ffi.Callback, abi ffi.Abi, nArgs uint32, rType *ffi.Type, argTypes ...*ffi.Type) (*callback, error) {
	cl, cb, err := getClosure(ffiCb, abi, nArgs, rType, argTypes...)
	if err != nil {
		return nil, err
	}

	return &callback{cl, cb}, nil
}

func (c *callback) getCallBackPtr() *unsafe.Pointer {
	return &c.cb
}

func (c *callback) free() {
	releaseClosure(c.cl)
	c.cb = nil
}

type nativeCallbacks struct {
	adapterEventCb unsafe.Pointer
	deviceEventCb  unsafe.Pointer
	authEventCb    unsafe.Pointer
	obexEventCb    unsafe.Pointer
}

type eventCallbacks struct {
	adapterEventCb *callback
	deviceEventCb  *callback
	authEventCb    *callback
	obexEventCb    *callback
}

func newEventCallbacks() (*eventCallbacks, error) {
	e := &eventCallbacks{}

	return e, e.setup()
}

func (e *eventCallbacks) toNativeCallbacks() *nativeCallbacks {
	return &nativeCallbacks{
		adapterEventCb: e.adapterEventCb.cb,
		deviceEventCb:  e.deviceEventCb.cb,
		authEventCb:    e.authEventCb.cb,
		obexEventCb:    e.obexEventCb.cb,
	}
}

func (e *eventCallbacks) free() {
	e.adapterEventCb.free()
	e.deviceEventCb.free()
	e.authEventCb.free()
	e.obexEventCb.free()
}

func (e *eventCallbacks) setup() error {
	var err error

	e.adapterEventCb, err = newCallback(func(cif *ffi.Cif, _ unsafe.Pointer, args *unsafe.Pointer, _ unsafe.Pointer) uintptr {
		eventArgs := unsafe.Slice(args, cif.NArgs)

		argEventAction := eventArgs[0]
		argAdapterData := eventArgs[1]

		eventAction := *(*nativeEventAction)(argEventAction)
		adapterData := *(**adapterNative)(argAdapterData)

		data := adapterData.toAdapterData()

		switch eventAction.ToEventAction() {
		case bluetooth.EventActionAdded:
			bluetooth.AdapterEvents().PublishAdded(data)

		case bluetooth.EventActionUpdated:
			bluetooth.AdapterEvents().PublishUpdated(data.AdapterEventData)

		case bluetooth.EventActionRemoved:
			bluetooth.AdapterEvents().PublishRemoved(data.AdapterEventData)
		}

		return 0
	}, ffi.DefaultAbi, 2, &ffi.TypeVoid, &ffi.TypeUint32, &ffi.TypePointer)
	if err != nil {
		return err
	}

	e.deviceEventCb, err = newCallback(func(cif *ffi.Cif, _ unsafe.Pointer, args *unsafe.Pointer, _ unsafe.Pointer) uintptr {
		eventArgs := unsafe.Slice(args, cif.NArgs)

		argEventAction := eventArgs[0]
		argDeviceData := eventArgs[1]

		eventAction := *(*nativeEventAction)(argEventAction)
		deviceData := *(**deviceNative)(argDeviceData)

		data := deviceData.ToDeviceData()

		switch eventAction.ToEventAction() {
		case bluetooth.EventActionAdded:
			bluetooth.DeviceEvents().PublishAdded(data)

		case bluetooth.EventActionUpdated:
			bluetooth.DeviceEvents().PublishUpdated(data.DeviceEventData)

		case bluetooth.EventActionRemoved:
			bluetooth.DeviceEvents().PublishRemoved(data.DeviceEventData)
		}

		return 0
	}, ffi.DefaultAbi, 2, &ffi.TypeVoid, &ffi.TypeUint32, &ffi.TypePointer)
	if err != nil {
		return err
	}

	e.authEventCb, err = newCallback(func(cif *ffi.Cif, _ unsafe.Pointer, args *unsafe.Pointer, _ unsafe.Pointer) uintptr {
		eventArgs := unsafe.Slice(args, cif.NArgs)

		argAuthEventType := eventArgs[0]
		argRequest := eventArgs[1]

		eventType := *(*authEventType)(argAuthEventType)
		handleAuthEvent(eventType, argRequest)

		return 0
	}, ffi.DefaultAbi, 2, &ffi.TypeVoid, &ffi.TypeUint32, &ffi.TypePointer)
	if err != nil {
		return err
	}

	e.obexEventCb, err = newCallback(func(cif *ffi.Cif, _ unsafe.Pointer, args *unsafe.Pointer, _ unsafe.Pointer) uintptr {
		eventArgs := unsafe.Slice(args, cif.NArgs)

		argObexEventType := eventArgs[0]
		argEventAction := eventArgs[1]
		argObexData := eventArgs[2]

		obexEventType := *(*obexProfile)(argObexEventType)
		eventAction := *(*nativeEventAction)(argEventAction)

		handleObexEvent(obexEventType, eventAction, argObexData)

		return 0
	}, ffi.DefaultAbi, 3, &ffi.TypeVoid, &ffi.TypeUint32, &ffi.TypeUint32, &ffi.TypePointer)
	if err != nil {
		return err
	}

	return nil
}

func getClosure(ffiCb ffi.Callback, abi ffi.Abi, nArgs uint32, rType *ffi.Type, argTypes ...*ffi.Type) (*ffi.Closure, unsafe.Pointer, error) {
	var cb unsafe.Pointer

	if ffiCb == nil {
		return nil, cb, errors.New("no FFI callback defined")
	}

	closure := ffi.ClosureAlloc(unsafe.Sizeof(ffi.Closure{}), &cb)
	if closure == nil {
		return nil, cb, errors.New("could not allocate closure")
	}

	var cif ffi.Cif
	if status := ffi.PrepCif(&cif, abi, nArgs, rType, argTypes...); status != ffi.OK {
		return nil, cb, errors.New("could not create closure CIF")
	}

	fn := ffi.NewCallback(ffiCb)

	if status := ffi.PrepClosureLoc(closure, &cif, fn, nil, cb); status != ffi.OK {
		return nil, cb, errors.New("could not prep closure")
	}

	return closure, cb, nil
}

func releaseClosure(closure *ffi.Closure) {
	if closure == nil {
		return
	}

	ffi.ClosureFree(closure)
}
