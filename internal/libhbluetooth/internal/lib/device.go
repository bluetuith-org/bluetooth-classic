//go:build !linux && libhbluetooth

package lib

import (
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/optional"
	ffi "github.com/bluetuith-org/libffi-go"
	"github.com/google/uuid"
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

	UUIDs     *uuid.UUID
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

	_hbcDeviceGetProperties.Call(libErr.getReturnPtr(), &argDeviceID, &argDeviceNative, libErr.getHbErrorPtr())
	if err := libErr.getError(); err != nil {
		var device bluetooth.DeviceData

		return device, err
	}
	defer deviceFree(&argDeviceNative)

	return argDeviceNative.ToDeviceData(), nil
}

// DeviceConnect connects a device on the associated adapter.
func DeviceConnect(address bluetooth.DeviceAddress) error {
	return deviceOperation(address, _hbcDeviceConnect)
}

// DeviceDisconnect disconnects a device from the associated adapter.
func DeviceDisconnect(address bluetooth.DeviceAddress) error {
	return deviceOperation(address, _hbcDeviceDisconnect)
}

// DevicePair pairs a device on the associated adapter.
func DevicePair(address bluetooth.DeviceAddress) error {
	var timeout int32
	return deviceOperation(address, _hbcDevicePair, &timeout)
}

// DevicePairCancel cancels a pairing request for the device.
func DevicePairCancel(address bluetooth.DeviceAddress) error {
	return deviceOperation(address, _hbcDevicePairCancel)
}

// DeviceRemove removes a device from the associated adapter.
func DeviceRemove(address bluetooth.DeviceAddress) error {
	return deviceOperation(address, _hbcDeviceRemove)
}

func deviceOperation(address bluetooth.DeviceAddress, opFun ffi.Fun, param ...any) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(address)

	if param == nil {
		opFun.Call(libErr.getReturnPtr(), &argDeviceID, libErr.getHbErrorPtr())
		return libErr.getError()
	}

	opFun.Call(libErr.getReturnPtr(), &argDeviceID, param[0], libErr.getHbErrorPtr())

	return libErr.getError()
}

func deviceFree(deviceData **deviceNative) {
	_hbcDeviceFree.Call(nil, deviceData)
}

var (
	_hbcDeviceGetProperties ffi.Fun
	_hbcDeviceFree          ffi.Fun
	_hbcDeviceIteratorFree  ffi.Fun

	_hbcDeviceConnect, _hbcDeviceDisconnect, _hbcDevicePair, _hbcDevicePairCancel, _hbcDeviceRemove ffi.Fun
)

func getDeviceFunHandles() []funHandle {
	return []funHandle{
		{
			&_hbcDeviceGetProperties, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_get_device", &fnRetType, &ffi.TypePointer, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcDeviceFree, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_device_free", &fnRetType, &ffi.TypePointer)
			},
		},
		{
			&_hbcDeviceIteratorFree, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_device_iterator_free", &fnRetType, &ffi.TypePointer)
			},
		},
		{
			&_hbcDeviceConnect, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_device_connect", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcDeviceDisconnect, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_device_disconnect", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcDevicePair, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_device_pair", &fnRetType, &ffi.TypePointer, &ffi.TypeSint32, &fnErrType)
			},
		},
		{
			&_hbcDevicePairCancel, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_device_pair_cancel", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcDeviceRemove, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_device_remove", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
	}
}
