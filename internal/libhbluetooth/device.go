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
	"github.com/bluetuith-org/bluetooth-classic/internal/libhbluetooth/internal/lib"
	"github.com/google/uuid"
)

type device struct {
	s   *BluetoothLibrary
	key bluetooth.DeviceAddress
}

// Pair will attempt to pair a bluetooth device that is in pairing mode.
func (d *device) Pair() error {
	if _, err := d.check(); err != nil {
		return err
	}

	return lib.DevicePair(d.key)
}

// CancelPairing will cancel a pairing attempt.
func (d *device) CancelPairing() error {
	if _, err := d.check(); err != nil {
		return err
	}

	return lib.DevicePairCancel(d.key)
}

// Connect will attempt to connect an already paired bluetooth device
// to an adapter.
func (d *device) Connect() error {
	if _, err := d.check(); err != nil {
		return err
	}

	return lib.DeviceConnect(d.key)
}

// Disconnect will disconnect the bluetooth device from the adapter.
func (d *device) Disconnect() error {
	if _, err := d.check(); err != nil {
		return err
	}

	return lib.DeviceDisconnect(d.key)
}

// ConnectProfile will attempt to connect an already paired bluetooth device
// to an adapter, using a specific Bluetooth profile UUID .
func (d *device) ConnectProfile(_ uuid.UUID) error {
	return errorkinds.ErrNotSupported
}

// DisconnectProfile will attempt to disconnect an already paired bluetooth device
// to an adapter, using a specific Bluetooth profile UUID .
func (d *device) DisconnectProfile(_ uuid.UUID) error {
	return errorkinds.ErrNotSupported
}

// Remove removes a device from its associated adapter.
func (d *device) Remove() error {
	if _, err := d.check(); err != nil {
		return err
	}

	return lib.DeviceRemove(d.key)
}

// SetTrusted sets the device 'trust' status within its associated adapter.
// Currently is valid only on Linux.
func (d *device) SetTrusted(_ bool) error {
	return errorkinds.ErrNotSupported
}

// SetBlocked sets the device 'blocked' status within its associated adapter.
// Currently is valid only on Linux.
func (d *device) SetBlocked(_ bool) error {
	return errorkinds.ErrNotSupported
}

// Properties returns all the properties of the device.
func (d *device) Properties() (bluetooth.DeviceData, error) {
	return d.check()
}

func (d *device) check() (bluetooth.DeviceData, error) {
	if d.s == nil || d.s.sessionClosed.Load() {
		return bluetooth.DeviceData{}, fault.Wrap(
			errorkinds.ErrSessionNotExist,
			fctx.With(
				context.Background(),
				"error_at", "device-check-bus",
				"address", d.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Error while fetching device data"),
		)
	}

	device, err := d.s.store.Device(d.key)
	if err != nil {
		return device, fault.Wrap(
			err,
			fctx.With(
				context.Background(),
				"error_at", "device-check-store",
				"address", d.key.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Adapter does not exist"),
		)
	}

	return device, nil
}
