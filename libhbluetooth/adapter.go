//go:build !linux && libhbluetooth

package libhbluetooth

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/bluetuith-org/bluetooth-classic/libhbluetooth/internal/lib"
)

type adapter struct {
	s   *BluetoothLibrary
	key bluetooth.AdapterAddress
}

// StartDiscovery will put the adapter into "discovering" mode, which means
// the bluetooth device will be able to discover other bluetooth devices
// that are in pairing mode.
func (a *adapter) StartDiscovery() error {
	if _, err := a.check(); err != nil {
		return err
	}

	return lib.AdapterStartDiscovery(a.key)
}

// StopDiscovery will stop the  "discovering" mode, which means the bluetooth device will
// no longer be able to discover other bluetooth devices that are in pairing mode.
func (a *adapter) StopDiscovery() error {
	if _, err := a.check(); err != nil {
		return err
	}

	return lib.AdapterStopDiscovery(a.key)
}

// SetPoweredState sets the powered state of the adapter.
func (a *adapter) SetPoweredState(enable bool) error {
	if _, err := a.check(); err != nil {
		return err
	}

	return lib.SetAdapterPoweredState(a.key, enable)
}

// SetDiscoverableState sets the discoverable state of the adapter.
func (a *adapter) SetDiscoverableState(enable bool) error {
	if _, err := a.check(); err != nil {
		return err
	}

	return lib.SetAdapterDiscoverableState(a.key, enable)
}

// SetPairableState sets the pairable state of the adapter.
func (a *adapter) SetPairableState(enable bool) error {
	if _, err := a.check(); err != nil {
		return err
	}

	return lib.SetAdapterPairableState(a.key, enable)
}

// Properties returns all the properties of the adapter.
func (a *adapter) Properties() (bluetooth.AdapterData, error) {
	return a.check()
}

// Devices returns all the devices associated with the adapter
func (a *adapter) Devices() ([]bluetooth.DeviceData, error) {
	if _, err := a.check(); err != nil {
		return nil, err
	}

	return a.s.store.AdapterDevices(a.key)
}

// check validates whether the adapter properties are present within the global session store.
func (a *adapter) check() (bluetooth.AdapterData, error) {
	if a.s == nil || a.s.sessionClosed.Load() {
		return bluetooth.AdapterData{}, fault.Wrap(
			errorkinds.ErrSessionNotExist,
			fctx.With(
				context.Background(),
				"error_at", "adapter-check-bus",
				"address", a.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Error while fetching adapter data"),
		)
	}

	adapter, err := a.s.store.Adapter(a.key)
	if err != nil {
		return adapter, fault.Wrap(
			err,
			fctx.With(
				context.Background(),
				"error_at", "adapter-check-store",
				"address", a.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Adapter does not exist"),
		)
	}

	return adapter, nil
}
