//go:build !linux && libhbluetooth

package lib

import (
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/optional"
	ffi "github.com/bluetuith-org/libffi-go"
	"github.com/google/uuid"
)

type adapterPropAttributes propAttributes

const (
	propIsPowered adapterPropAttributes = 1 << iota
	propIsDiscoverable
	propIsPairable
	propIsDiscovering
)

type adapterNative struct {
	Address bdAddr

	UUIDs     *uuid.UUID
	UUIDCount uint32

	Name       *byte
	Alias      *byte
	UniqueName *byte

	Attributes     uint32
	IsPowered      bool
	IsDiscoverable bool
	IsPairable     bool
	IsDiscovering  bool
}

func newAdapterNative() *adapterNative {
	return &adapterNative{}
}

func (a *adapterNative) toAdapterData() bluetooth.AdapterData {
	adapter := bluetooth.AdapterData{
		UniqueName: bytePtrToString(a.UniqueName),
		AdapterEventData: bluetooth.AdapterEventData{
			AdapterAddress: bluetooth.AdapterAddress{
				Address: a.Address.Data,
			},

			UUIDs: ptrToUUIDs(a.UUIDs, a.UUIDCount),
		},
	}

	checkAndSetAttrs(propIsPowered, a.Attributes, optSetFunc(&adapter.Powered, a.IsPowered))
	checkAndSetAttrs(propIsDiscoverable, a.Attributes, optSetFunc(&adapter.Discoverable, a.IsDiscoverable))
	checkAndSetAttrs(propIsPairable, a.Attributes, optSetFunc(&adapter.Pairable, a.IsPairable))
	checkAndSetAttrs(propIsDiscovering, a.Attributes, optSetFunc(&adapter.Discovering, a.IsDiscovering))

	name := bytePtrToString(a.Name)
	if name != "" {
		adapter.Name = optional.New(name)
	}

	alias := bytePtrToString(a.Alias)
	if alias != "" {
		adapter.Alias = optional.New(alias)
	}

	return adapter
}

// GetAdapters returns a list of available adapters.
func GetAdapters() ([]bluetooth.AdapterData, error) {
	libErr := newLibError()

	argNativeArray := newNativeArray[adapterNative]()

	_hbcGetAdapters.Call(libErr.getReturnPtr(), &argNativeArray, libErr.getHbErrorPtr())
	if err := libErr.getError(); err != nil {
		return nil, err
	}
	if argNativeArray.Count == 0 || argNativeArray.List == nil {
		return nil, nil
	}
	defer argNativeArray.free(_hbcAdapterIteratorFree)

	arrv := unsafe.Slice((**adapterNative)(unsafe.Pointer(argNativeArray.List)), argNativeArray.Count)
	adapters := make([]bluetooth.AdapterData, 0, argNativeArray.Count)
	for _, v := range arrv {
		adapters = append(adapters, v.toAdapterData())
	}

	return adapters, nil
}

// AdapterGetPairedDevices returns a list of paired devices on the current adapter.
func AdapterGetPairedDevices(address bluetooth.AdapterAddress) ([]bluetooth.DeviceData, error) {
	libErr := newLibError()

	argNativeArray := newNativeArray[deviceNative]()
	argBdaddr := newBdAddr(address.Address)

	_hbcAdapterGetDevices.Call(libErr.getReturnPtr(), &argBdaddr, &argNativeArray, libErr.getHbErrorPtr())
	if err := libErr.getError(); err != nil {
		return nil, err
	}
	if argNativeArray.Count == 0 || argNativeArray.List == nil {
		return nil, nil
	}
	defer argNativeArray.free(_hbcDeviceIteratorFree)

	arrv := unsafe.Slice((**deviceNative)(unsafe.Pointer(argNativeArray.List)), argNativeArray.Count)
	devices := make([]bluetooth.DeviceData, 0, argNativeArray.Count)
	for _, v := range arrv {
		devices = append(devices, v.ToDeviceData())
	}

	return devices, nil
}

// AdapterProperties returns the full information of the specified adapter.
func AdapterProperties(address bluetooth.AdapterAddress) (bluetooth.AdapterData, error) {
	libErr := newLibError()

	argBdaddr := newBdAddr(address.Address)
	argAdapter := newAdapterNative()

	_hbcAdapterGetProperties.Call(libErr.getReturnPtr(), &argBdaddr, &argAdapter, libErr.getHbErrorPtr())
	if err := libErr.getError(); err != nil {
		var adapter bluetooth.AdapterData

		return adapter, err
	}
	defer adapterFree(&argAdapter)

	return argAdapter.toAdapterData(), nil
}

// AdapterStartDiscovery starts a device discovery on the specified adapter.
// All found devices will be published as events. Look for device events with "paired: false" and "event action: Added".
func AdapterStartDiscovery(address bluetooth.AdapterAddress) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)
	argTimeout := int32(0)

	_hbcAdapterStartDiscovery.Call(libErr.getReturnPtr(), &argBdAddr, &argTimeout, libErr.getHbErrorPtr())

	return libErr.getError()
}

// AdapterStopDiscovery stops the device discovery on the specified adapter.
func AdapterStopDiscovery(address bluetooth.AdapterAddress) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)

	_hbcAdapterStopDiscovery.Call(libErr.getReturnPtr(), &argBdAddr, libErr.getHbErrorPtr())

	return libErr.getError()
}

// SetAdapterPoweredState sets the powered state for the adapter.
func SetAdapterPoweredState(address bluetooth.AdapterAddress, state bool) error {
	return adapterSetState(address, state, &_hbcSetPoweredState)
}

// SetAdapterDiscoverableState sets the discoverable state for the adapter.
func SetAdapterDiscoverableState(address bluetooth.AdapterAddress, state bool) error {
	return adapterSetState(address, state, &_hbcSetDiscoverableState)
}

// SetAdapterPairableState sets the pairable state for the adapter.
func SetAdapterPairableState(address bluetooth.AdapterAddress, state bool) error {
	return adapterSetState(address, state, &_hbcSetPairableState)
}

func adapterSetState(address bluetooth.AdapterAddress, state bool, fun *ffi.Fun) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)

	fun.Call(libErr.getReturnPtr(), &argBdAddr, &state, libErr.getHbErrorPtr())
	return libErr.getError()
}

func adapterFree(an **adapterNative) {
	_hbcAdapterFree.Call(nil, an)
}

var (
	_hbcAdapterGetProperties ffi.Fun
	_hbcAdapterFree          ffi.Fun

	_hbcAdapterGetDevices ffi.Fun

	_hbcGetAdapters         ffi.Fun
	_hbcAdapterIteratorFree ffi.Fun

	_hbcAdapterStartDiscovery ffi.Fun
	_hbcAdapterStopDiscovery  ffi.Fun

	_hbcSetPoweredState, _hbcSetDiscoverableState, _hbcSetPairableState ffi.Fun
)

func getAdapterFunHandles() []funHandle {
	return []funHandle{
		{
			&_hbcAdapterGetProperties, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_get_adapter", &fnRetType, &ffi.TypePointer, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcAdapterFree, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_free", &ffi.TypeVoid, &ffi.TypePointer)
			},
		},
		{
			&_hbcGetAdapters, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_get_adapters", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcAdapterIteratorFree, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_iterator_free", &ffi.TypeVoid, &ffi.TypePointer)
			},
		},
		{
			&_hbcAdapterGetDevices, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_get_paired_devices", &fnRetType, &ffi.TypePointer, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcAdapterStartDiscovery, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_start_discovery", &fnRetType, &ffi.TypePointer, &ffi.TypeSint32, &fnErrType)
			},
		},
		{
			&_hbcAdapterStopDiscovery, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_stop_discovery", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcSetPoweredState, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_set_powered_state", &fnRetType, &ffi.TypePointer, &ffi.TypeUint32, &fnErrType)
			},
		},
		{
			&_hbcSetDiscoverableState, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_set_discoverable_state", &fnRetType, &ffi.TypePointer, &ffi.TypeUint32, &fnErrType)
			},
		},
		{
			&_hbcSetPairableState, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_adapter_set_pairable_state", &fnRetType, &ffi.TypePointer, &ffi.TypeUint32, &fnErrType)
			},
		},
	}
}
