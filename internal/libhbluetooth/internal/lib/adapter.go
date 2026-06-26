//go:build !linux && libhbluetooth

package lib

import (
	"runtime"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/optional"
	"github.com/ebitengine/purego"
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

	UUIDs     *guid
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

	ret := _hbcGetAdapters.Call(argNativeArray, libErr.getHbErrorPtr())
	if err := libErr.getError(ret); err != nil {
		return nil, err
	}
	if argNativeArray.Count == 0 || argNativeArray.List == nil {
		return nil, nil
	}
	defer _hbcAdapterIteratorFree.Call(argNativeArray)

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

	ret := _hbcAdapterGetDevices.Call(argBdaddr, argNativeArray, libErr.getHbErrorPtr())
	if err := libErr.getError(ret); err != nil {
		return nil, err
	}
	if argNativeArray.Count == 0 || argNativeArray.List == nil {
		return nil, nil
	}
	defer _hbcDeviceIteratorFree.Call(argNativeArray)

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

	ret := _hbcAdapterGetProperties.Call(argBdaddr, argAdapter, libErr.getHbErrorPtr())
	if err := libErr.getError(ret); err != nil {
		var adapter bluetooth.AdapterData

		return adapter, err
	}
	defer _hbcAdapterFree.Call(argAdapter)

	return argAdapter.toAdapterData(), nil
}

// AdapterStartDiscovery starts a device discovery on the specified adapter.
// All found devices will be published as events. Look for device events with "paired: false" and "event action: Added".
func AdapterStartDiscovery(address bluetooth.AdapterAddress) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)
	argTimeout := int32(0)

	ret := _hbcAdapterStartDiscovery.Call(argBdAddr, argTimeout, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// AdapterStopDiscovery stops the device discovery on the specified adapter.
func AdapterStopDiscovery(address bluetooth.AdapterAddress) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)
	ret := _hbcAdapterStopDiscovery.Call(argBdAddr, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// SetAdapterPoweredState sets the powered state for the adapter.
func SetAdapterPoweredState(address bluetooth.AdapterAddress, state bool) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)
	ret := _hbcSetPoweredState.Call(argBdAddr, state, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// SetAdapterDiscoverableState sets the discoverable state for the adapter.
func SetAdapterDiscoverableState(address bluetooth.AdapterAddress, state bool) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)
	ret := _hbcSetDiscoverableState.Call(argBdAddr, state, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// SetAdapterPairableState sets the pairable state for the adapter.
func SetAdapterPairableState(address bluetooth.AdapterAddress, state bool) error {
	libErr := newLibError()

	argBdAddr := newBdAddr(address.Address)
	ret := _hbcSetPairableState.Call(argBdAddr, state, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

var (
	_hbcAdapterGetProperties interopFunc[func(*bdAddr, *adapterNative, **hbError) hbStatus]
	_hbcAdapterFree          interopFunc[func(*adapterNative)]

	_hbcAdapterGetDevices interopFunc[func(*bdAddr, *nativeArray[deviceNative], **hbError) hbStatus]

	_hbcGetAdapters         interopFunc[func(*nativeArray[adapterNative], **hbError) hbStatus]
	_hbcAdapterIteratorFree interopFunc[func(*nativeArray[adapterNative])]

	_hbcAdapterStartDiscovery interopFunc[func(*bdAddr, int32, **hbError) hbStatus]
	_hbcAdapterStopDiscovery  interopFunc[func(*bdAddr, **hbError) hbStatus]

	_hbcSetPoweredState, _hbcSetDiscoverableState, _hbcSetPairableState interopFunc[func(*bdAddr, bool, **hbError) hbStatus]
)

func getAdapterFunHandles() []funHandle {
	return []funHandle{
		newInteropFunc("hbc_get_adapter", &_hbcAdapterGetProperties, func(addr *bdAddr, data *adapterNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcAdapterGetProperties.funAddr(), uintptr(unsafe.Pointer(addr)), uintptr(unsafe.Pointer(data)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(data)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_adapter_free", &_hbcAdapterFree, func(data *adapterNative) {
			_, _, _ = purego.SyscallN(_hbcAdapterFree.funAddr(), uintptr(unsafe.Pointer(data)))
			runtime.KeepAlive(data)
		}),
		newInteropFunc("hbc_get_adapters", &_hbcGetAdapters, func(iterator *nativeArray[adapterNative], hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcGetAdapters.funAddr(), uintptr(unsafe.Pointer(iterator)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(iterator)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_adapter_iterator_free", &_hbcAdapterIteratorFree, func(iterator *nativeArray[adapterNative]) {
			_, _, _ = purego.SyscallN(_hbcAdapterIteratorFree.funAddr(), uintptr(unsafe.Pointer(iterator)))
			runtime.KeepAlive(iterator)
		}),
		newInteropFunc("hbc_adapter_get_paired_devices", &_hbcAdapterGetDevices, func(addr *bdAddr, iterator *nativeArray[deviceNative], hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcAdapterGetDevices.funAddr(), uintptr(unsafe.Pointer(addr)), uintptr(unsafe.Pointer(iterator)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(iterator)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_adapter_start_discovery", &_hbcAdapterStartDiscovery, func(addr *bdAddr, timeout int32, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcAdapterStartDiscovery.funAddr(), uintptr(unsafe.Pointer(addr)), uintptr(timeout), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_adapter_stop_discovery", &_hbcAdapterStopDiscovery, func(addr *bdAddr, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcAdapterStopDiscovery.funAddr(), uintptr(unsafe.Pointer(addr)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_adapter_set_powered_state", &_hbcSetPoweredState, func(addr *bdAddr, state bool, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcSetPoweredState.funAddr(), uintptr(unsafe.Pointer(addr)), boolToUintptr(state), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_adapter_set_discoverable_state", &_hbcSetDiscoverableState, func(addr *bdAddr, state bool, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcSetDiscoverableState.funAddr(), uintptr(unsafe.Pointer(addr)), boolToUintptr(state), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_adapter_set_pairable_state", &_hbcSetPairableState, func(addr *bdAddr, state bool, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcSetPairableState.funAddr(), uintptr(unsafe.Pointer(addr)), boolToUintptr(state), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
	}
}

func boolToUintptr(v bool) uintptr {
	if v {
		return 1
	}

	return 0
}
