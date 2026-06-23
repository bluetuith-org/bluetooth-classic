//go:build !linux && haraltd

package haraltd

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/bluetuith-org/bluetooth-classic/internal/haraltd/internal/commands"
)

// adapter describes a function call interface to invoke adapter related functions.
type adapter struct {
	s   *HaraltdSession
	key bluetooth.AdapterAddress
}

// StartDiscovery will put the adapter into "discovering" mode, which means
// the bluetooth device will be able to discover other bluetooth devices
// that are in pairing mode.
func (a *adapter) StartDiscovery() error {
	if _, err := a.check(); err != nil {
		return err
	}

	_, err := commands.StartDiscovery(a.key.Address).ExecuteWith(a.s.executor)
	if err != nil {
		return fault.Wrap(
			err,
			fctx.With(
				context.Background(),
				"error_at", "adapter-start-discovery",
				"address", a.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("An error occurred while starting device discovery"),
		)
	}

	return nil
}

// StopDiscovery will stop the  "discovering" mode, which means the bluetooth device will
// no longer be able to discover other bluetooth devices that are in pairing mode.
func (a *adapter) StopDiscovery() error {
	if _, err := a.check(); err != nil {
		return err
	}

	_, err := commands.StopDiscovery(a.key.Address).ExecuteWith(a.s.executor)
	if err != nil {
		return fault.Wrap(
			err,
			fctx.With(
				context.Background(),
				"error_at", "adapter-stop-discovery",
				"address", a.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("An error occurred while stopping device discovery"),
		)
	}

	return nil
}

// SetPoweredState sets the powered state of the adapter.
func (a *adapter) SetPoweredState(enable bool) error {
	if _, err := a.check(); err != nil {
		return err
	}

	_, err := commands.SetPoweredState(a.key.Address, enable).ExecuteWith(a.s.executor)
	if err != nil {
		return fault.Wrap(
			err,
			fctx.With(
				context.Background(),
				"error_at", "adapter-setpowered-state",
				"address", a.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("An error occurred on setting powered state"),
		)
	}

	return nil
}

// SetDiscoverableState sets the discoverable state of the adapter.
func (a *adapter) SetDiscoverableState(enable bool) error {
	if _, err := a.check(); err != nil {
		return err
	}

	_, err := commands.SetDiscoverableState(a.key.Address, enable).ExecuteWith(a.s.executor)
	if err != nil {
		return fault.Wrap(
			err,
			fctx.With(
				context.Background(),
				"error_at", "adapter-setdiscoverable-state",
				"address", a.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("An error occurred on setting discoverable state"),
		)
	}

	return nil
}

// SetPairableState sets the pairable state of the adapter.
func (a *adapter) SetPairableState(enable bool) error {
	if _, err := a.check(); err != nil {
		return err
	}

	_, err := commands.SetPairableState(a.key.Address, enable).ExecuteWith(a.s.executor)
	if err != nil {
		return fault.Wrap(
			err,
			fctx.With(
				context.Background(),
				"error_at", "adapter-setpairable-state",
				"address", a.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("An error occurred on setting pairable state"),
		)
	}

	return nil
}

// Properties returns all the properties of the adapter.
func (a *adapter) Properties() (bluetooth.AdapterData, error) {
	return a.check()
}

// Devices returns all the devices associated with the adapter
func (a *adapter) Devices() ([]bluetooth.DeviceData, error) {
	_, err := a.check()
	if err != nil {
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

// appendProperties appends any additional properties to the provided adapter and returns
// the new result.
// It is currently a placeholder function only.
func (a *adapter) appendProperties(adapter bluetooth.AdapterData) (bluetooth.AdapterData, error) {
	return adapter, nil
}
