//go:build !linux && libhbluetooth

package lib

import (
	"runtime"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/optional"
	"github.com/ebitengine/purego"
)

type devicePropAttributes propAttributes

const (
	propIsConnectede devicePropAttributes = 1 << iota
	propIsPaired
	propHasRSSI
	propHasBatteryPercentage
)

type deviceIDNative struct {
	Address, AdapterAddress bdAddr
}

func newDeviceID(address bluetooth.DeviceAddress) *deviceIDNative {
	return &deviceIDNative{
		Address:        newBdAddrValue(address.Address),
		AdapterAddress: newBdAddrValue(address.AssociatedAdapter),
	}
}

func (d *deviceIDNative) ToDeviceAddress() bluetooth.DeviceAddress {
	return bluetooth.NewDeviceAddress(d.Address.Data, d.AdapterAddress.Data)
}

type deviceNative struct {
	id deviceIDNative

	UUIDs     *guid
	UUIDCount uint32

	Name  *byte
	Alias *byte

	Class         uint32
	LegacyPairing bool

	Attributes           uint32
	IsConnected          bool
	IsPaired             bool
	HasRSSI              int16
	HasBatteryPercentage uint32
}

func newDeviceNative() *deviceNative {
	return &deviceNative{}
}

func (d *deviceNative) ToDeviceData() bluetooth.DeviceData {
	device := bluetooth.DeviceData{
		Class:         d.Class,
		LegacyPairing: d.LegacyPairing,
		DeviceEventData: bluetooth.DeviceEventData{
			DeviceAddress: bluetooth.DeviceAddress{
				Address:           d.id.Address.Data,
				AssociatedAdapter: d.id.AdapterAddress.Data,
			},

			UUIDs: ptrToUUIDs(d.UUIDs, d.UUIDCount),
		},
	}

	name := bytePtrToString(d.Name)
	if name != "" {
		device.Name = optional.New(name)
	}

	alias := bytePtrToString(d.Alias)
	if alias != "" {
		device.Alias = optional.New(alias)
	}

	device.Type = bluetooth.DeviceTypeFromClass(d.Class)

	checkAndSetAttrs(propIsConnectede, d.Attributes, optSetFunc(&device.Connected, d.IsConnected))
	checkAndSetAttrs(propIsPaired, d.Attributes, optSetFunc(&device.Paired, d.IsPaired))
	checkAndSetAttrs(propHasRSSI, d.Attributes, optSetFunc(&device.RSSI, d.HasRSSI))
	checkAndSetAttrs(propHasBatteryPercentage, d.Attributes, optSetFunc(&device.Percentage, d.HasBatteryPercentage))

	return device
}

// DeviceProperties returns the entire device information for the provided device on its associated adapter.
func DeviceProperties(address bluetooth.DeviceAddress) (bluetooth.DeviceData, error) {
	libErr := newLibError()

	argDeviceID := newDeviceID(address)
	argDeviceNative := newDeviceNative()

	ret := _hbcDeviceGetProperties.Call(argDeviceID, argDeviceNative, libErr.getHbErrorPtr())
	if err := libErr.getError(ret); err != nil {
		var device bluetooth.DeviceData

		return device, err
	}
	defer _hbcDeviceFree.Call(argDeviceNative)

	return argDeviceNative.ToDeviceData(), nil
}

// DeviceConnect connects a device on the associated adapter.
func DeviceConnect(address bluetooth.DeviceAddress) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(address)
	ret := _hbcDeviceConnect.Call(argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// DeviceDisconnect disconnects a device from the associated adapter.
func DeviceDisconnect(address bluetooth.DeviceAddress) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(address)
	ret := _hbcDeviceDisconnect.Call(argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// DevicePair pairs a device on the associated adapter.
func DevicePair(address bluetooth.DeviceAddress) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(address)
	ret := _hbcDevicePair.Call(argDeviceID, 0, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// DevicePairCancel cancels a pairing request for the device.
func DevicePairCancel(address bluetooth.DeviceAddress) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(address)
	ret := _hbcDevicePairCancel.Call(argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// DeviceRemove removes a device from the associated adapter.
func DeviceRemove(address bluetooth.DeviceAddress) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(address)
	ret := _hbcDeviceRemove.Call(argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

var (
	_hbcDeviceGetProperties interopFunc[func(*deviceIDNative, *deviceNative, **hbError) hbStatus]
	_hbcDeviceFree          interopFunc[func(*deviceNative)]
	_hbcDeviceIteratorFree  interopFunc[func(*nativeArray[deviceNative])]
	_hbcDevicePair          interopFunc[func(*deviceIDNative, int32, **hbError) hbStatus]

	_hbcDeviceConnect, _hbcDeviceDisconnect, _hbcDevicePairCancel, _hbcDeviceRemove interopFunc[func(*deviceIDNative, **hbError) hbStatus]
)

func getDeviceFunHandles() []funHandle {
	return []funHandle{
		newInteropFunc("hbc_get_device", &_hbcDeviceGetProperties, func(id *deviceIDNative, data *deviceNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcDeviceGetProperties.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(data)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(data)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_device_free", &_hbcDeviceFree, func(data *deviceNative) {
			_, _, _ = purego.SyscallN(_hbcDeviceFree.funAddr(), uintptr(unsafe.Pointer(data)))
			runtime.KeepAlive(data)
		}),
		newInteropFunc("hbc_device_iterator_free", &_hbcDeviceIteratorFree, func(iterator *nativeArray[deviceNative]) {
			_, _, _ = purego.SyscallN(_hbcDeviceIteratorFree.funAddr(), uintptr(unsafe.Pointer(iterator)))
			runtime.KeepAlive(iterator)
		}),
		newInteropFunc("hbc_device_connect", &_hbcDeviceConnect, func(id *deviceIDNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcDeviceConnect.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_device_disconnect", &_hbcDeviceDisconnect, func(id *deviceIDNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcDeviceDisconnect.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_device_pair", &_hbcDevicePair, func(id *deviceIDNative, timeout int32, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcDevicePair.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(timeout), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_device_pair_cancel", &_hbcDevicePairCancel, func(id *deviceIDNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcDevicePairCancel.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_device_remove", &_hbcDeviceRemove, func(id *deviceIDNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcDeviceRemove.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(hberr)
			return ret
		}),
	}
}
