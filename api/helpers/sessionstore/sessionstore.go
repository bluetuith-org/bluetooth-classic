package sessionstore

import (
	"errors"
	"fmt"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/puzpuzpuz/xsync/v3"
)

// MergeAdapterDataFunc describes a function to merge old adapter data
// with updated adapter data.
type MergeAdapterDataFunc func(*bluetooth.AdapterData) error

// MergeDeviceDataFunc describes a function to merge old device data
// with updated device data.
type MergeDeviceDataFunc func(*bluetooth.DeviceData) error

// SessionStore describes a store of adapters and devices.
type SessionStore struct {
	adapters *xsync.MapOf[bluetooth.AdapterAddress, bluetooth.AdapterData]
	devices  *xsync.MapOf[bluetooth.DeviceAddress, bluetooth.DeviceData]
}

// NewSessionStore returns a new SessionStore.
func NewSessionStore() SessionStore {
	return SessionStore{
		adapters: xsync.NewMapOf[bluetooth.AdapterAddress, bluetooth.AdapterData](),
		devices:  xsync.NewMapOf[bluetooth.DeviceAddress, bluetooth.DeviceData](),
	}
}

// Adapters returns a list of adapters from the store.
func (s *SessionStore) Adapters() ([]bluetooth.AdapterData, error) {
	adapters := make([]bluetooth.AdapterData, 0, s.adapters.Size())

	s.adapters.Range(func(_ bluetooth.AdapterAddress, adapter bluetooth.AdapterData) bool {
		adapters = append(adapters, adapter)

		return true
	})

	if len(adapters) == 0 {
		return nil, errors.New("no adapters found")
	}

	return adapters, nil
}

// Adapter returns an adapter which matches the provided address.
func (s *SessionStore) Adapter(address bluetooth.AdapterAddress) (bluetooth.AdapterData, error) {
	adapter, ok := s.adapters.Load(address)
	if !ok {
		return adapter, fmt.Errorf("get %q: %w", address.Address.String(), errorkinds.ErrAdapterNotFound)
	}

	return adapter, nil
}

// AdapterDevices returns a list of devices that are associated with the specified adapter address.
func (s *SessionStore) AdapterDevices(address bluetooth.AdapterAddress) ([]bluetooth.DeviceData, error) {
	_, ok := s.adapters.Load(address)
	if !ok {
		return nil, fmt.Errorf("find %q: %w", address.Address.String(), errorkinds.ErrAdapterNotFound)
	}

	devices := make([]bluetooth.DeviceData, 0, s.devices.Size())
	s.devices.Range(func(_ bluetooth.DeviceAddress, d bluetooth.DeviceData) bool {
		if d.AssociatedAdapter == address.Address {
			devices = append(devices, d)
		}

		return true
	})

	return devices, nil
}

// AddAdapter adds an adapter to the store.
func (s *SessionStore) AddAdapter(adapter bluetooth.AdapterData) {
	s.adapters.Store(adapter.AdapterAddress, adapter)
}

// AddAdapters adds a list of adapters to the store.
func (s *SessionStore) AddAdapters(adapters ...bluetooth.AdapterData) {
	for _, adapter := range adapters {
		s.adapters.Store(adapter.AdapterAddress, adapter)
	}
}

// RemoveAdapter removes an adapter from the store.
func (s *SessionStore) RemoveAdapter(address bluetooth.AdapterAddress) {
	s.adapters.Delete(address)
}

// UpdateAdapter updates the properties of the adapter in the store.
func (s *SessionStore) UpdateAdapter(
	address bluetooth.AdapterAddress,
	mergefn MergeAdapterDataFunc,
) (bluetooth.AdapterEventData, error) {
	adapter, ok := s.adapters.Load(address)
	if !ok {
		return bluetooth.AdapterEventData{},
			fmt.Errorf("update %q: %w", address.Address.String(), errorkinds.ErrAdapterNotFound)
	}

	if err := mergefn(&adapter); err != nil {
		return bluetooth.AdapterEventData{}, err
	}

	s.adapters.Store(address, adapter)

	return adapter.AdapterEventData, nil
}

// Device returns a device which matches the provided address.
func (s *SessionStore) Device(address bluetooth.DeviceAddress) (bluetooth.DeviceData, error) {
	device, ok := s.devices.Load(address)
	if !ok {
		return bluetooth.DeviceData{},
			fmt.Errorf("get %q (adapter %q): %w", address.Address.String(), address.AssociatedAdapter.String(), errorkinds.ErrDeviceNotFound)
	}

	return device, nil
}

// AddDevice adds a device to the store.
func (s *SessionStore) AddDevice(device bluetooth.DeviceData) {
	s.devices.Store(device.DeviceAddress, device)
}

// AddDevices adds a list of devices to the store.
func (s *SessionStore) AddDevices(devices ...bluetooth.DeviceData) {
	for _, device := range devices {
		s.devices.Store(device.DeviceAddress, device)
	}
}

// RemoveDevice removes a device from the store.
func (s *SessionStore) RemoveDevice(address bluetooth.DeviceAddress) {
	s.devices.Delete(address)
}

// UpdateDevice updates the properties of the device in the store.
func (s *SessionStore) UpdateDevice(
	address bluetooth.DeviceAddress,
	mergefn MergeDeviceDataFunc,
) (bluetooth.DeviceEventData, error) {
	device, ok := s.devices.Load(address)
	if !ok {
		return bluetooth.DeviceEventData{},
			fmt.Errorf("update %q (adapter %q): %w", address.Address.String(), address.AssociatedAdapter.String(), errorkinds.ErrDeviceNotFound)
	}

	if err := mergefn(&device); err != nil {
		return bluetooth.DeviceEventData{}, err
	}

	s.devices.Store(address, device)

	return device.DeviceEventData, nil
}
