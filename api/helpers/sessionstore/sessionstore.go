package sessionstore

import (
	"fmt"
	"sync"
	"sync/atomic"

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
	adapters *xsync.MapOf[bluetooth.MacAddress, bluetooth.AdapterData]
	devices  *xsync.MapOf[bluetooth.MacAddress, bluetooth.DeviceData]

	init    sync.WaitGroup
	waiting atomic.Bool
}

// NewSessionStore returns a new SessionStore.
func NewSessionStore() SessionStore {
	return SessionStore{
		adapters: xsync.NewMapOf[bluetooth.MacAddress, bluetooth.AdapterData](),
		devices:  xsync.NewMapOf[bluetooth.MacAddress, bluetooth.DeviceData](),
	}
}

// WaitInitialize waits for the store to be initialized.
// When this is called, reading or updating existing values in the store,
// using functions like Adapter(), UpdateAdapter() will be paused and only
// populating the store (via AddAdapter(), AddDevice() etc) can be done.
// This should be used to queue any updates to the store while populating it.
func (s *SessionStore) WaitInitialize() {
	if s.waiting.Load() {
		return
	}

	s.init.Add(1)
	s.waiting.Store(true)
}

// DoneInitialize enables all reading/updation operations to be resumed after store initialization.
func (s *SessionStore) DoneInitialize() {
	s.init.Done()
	s.waiting.Store(false)
}

// Adapters returns a list of adapters from the store.
func (s *SessionStore) Adapters() []bluetooth.AdapterData {
	s.init.Wait()

	adapters := make([]bluetooth.AdapterData, 0, s.adapters.Size())

	s.adapters.Range(func(_ bluetooth.MacAddress, adapter bluetooth.AdapterData) bool {
		adapters = append(adapters, adapter)

		return true
	})

	return adapters
}

// Adapter returns an adapter which matches the provided address.
func (s *SessionStore) Adapter(adapterAddress bluetooth.MacAddress) (bluetooth.AdapterData, error) {
	s.init.Wait()

	adapter, ok := s.adapters.Load(adapterAddress)
	if !ok {
		return adapter, fmt.Errorf("get %q: %w", adapterAddress.String(), errorkinds.ErrAdapterNotFound)
	}

	return adapter, nil
}

// AdapterDevices returns a list of devices that are associated with the specified adapter address.
func (s *SessionStore) AdapterDevices(adapterAddress bluetooth.MacAddress) ([]bluetooth.DeviceData, error) {
	s.init.Wait()

	_, ok := s.adapters.Load(adapterAddress)
	if !ok {
		return nil, fmt.Errorf("find %q: %w", adapterAddress.String(), errorkinds.ErrAdapterNotFound)
	}

	devices := make([]bluetooth.DeviceData, 0, s.devices.Size())
	s.devices.Range(func(_ bluetooth.MacAddress, d bluetooth.DeviceData) bool {
		if d.AssociatedAdapter == adapterAddress {
			devices = append(devices, d)
		}

		return true
	})

	return devices, nil
}

// AddAdapter adds an adapter to the store.
func (s *SessionStore) AddAdapter(adapter bluetooth.AdapterData) {
	s.adapters.Store(adapter.Address, adapter)
}

// AddAdapters adds a list of adapters to the store.
func (s *SessionStore) AddAdapters(adapters ...bluetooth.AdapterData) {
	for _, adapter := range adapters {
		s.adapters.Store(adapter.Address, adapter)
	}
}

// RemoveAdapter removes an adapter from the store.
func (s *SessionStore) RemoveAdapter(adapterAddress bluetooth.MacAddress) {
	s.adapters.Delete(adapterAddress)
}

// UpdateAdapter updates the properties of the adapter in the store.
func (s *SessionStore) UpdateAdapter(
	adapterAddress bluetooth.MacAddress,
	mergefn MergeAdapterDataFunc,
) (bluetooth.AdapterEventData, error) {
	s.init.Wait()

	adapter, ok := s.adapters.Load(adapterAddress)
	if !ok {
		return bluetooth.AdapterEventData{},
			fmt.Errorf("update %q: %w", adapterAddress.String(), errorkinds.ErrAdapterNotFound)
	}

	if err := mergefn(&adapter); err != nil {
		return bluetooth.AdapterEventData{}, err
	}

	s.adapters.Store(adapterAddress, adapter)

	return adapter.AdapterEventData, nil
}

// Device returns a device which matches the provided address.
func (s *SessionStore) Device(deviceAddress bluetooth.MacAddress) (bluetooth.DeviceData, error) {
	s.init.Wait()

	device, ok := s.devices.Load(deviceAddress)
	if !ok {
		return bluetooth.DeviceData{},
			fmt.Errorf("get %q: %w", deviceAddress.String(), errorkinds.ErrDeviceNotFound)
	}

	return device, nil
}

// AddDevice adds a device to the store.
func (s *SessionStore) AddDevice(device bluetooth.DeviceData) {
	s.devices.Store(device.Address, device)
}

// AddDevices adds a list of devices to the store.
func (s *SessionStore) AddDevices(devices ...bluetooth.DeviceData) {
	for _, device := range devices {
		s.devices.Store(device.Address, device)
	}
}

// RemoveDevice removes a device from the store.
func (s *SessionStore) RemoveDevice(deviceAddress bluetooth.MacAddress) {
	s.devices.Delete(deviceAddress)
}

// UpdateDevice updates the properties of the device in the store.
func (s *SessionStore) UpdateDevice(
	deviceAddress bluetooth.MacAddress,
	mergefn MergeDeviceDataFunc,
) (bluetooth.DeviceEventData, error) {
	s.init.Wait()

	device, ok := s.devices.Load(deviceAddress)
	if !ok {
		return bluetooth.DeviceEventData{},
			fmt.Errorf("update %q: %w", deviceAddress.String(), errorkinds.ErrDeviceNotFound)
	}

	if err := mergefn(&device); err != nil {
		return bluetooth.DeviceEventData{}, err
	}

	s.devices.Store(deviceAddress, device)

	return device.DeviceEventData, nil
}
