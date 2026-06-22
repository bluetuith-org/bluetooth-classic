//go:build !linux && libhbluetooth

package libhbluetooth

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/config"
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/bluetuith-org/bluetooth-classic/api/platforminfo"
	"github.com/bluetuith-org/bluetooth-classic/libhbluetooth/internal/lib"

	ac "github.com/bluetuith-org/bluetooth-classic/api/appfeatures"
	sstore "github.com/bluetuith-org/bluetooth-classic/api/helpers/sessionstore"
)

const implementation = "libhbluetooth"

// BluetoothLibrary describes an interface to interact with libhbluetooth.
type BluetoothLibrary struct {
	features   *ac.FeatureSet
	authorizer bluetooth.SessionAuthorizer

	sessionClosed atomic.Bool
	store         sstore.SessionStore

	sync.Mutex
}

// Start attempts to initialize a session with the system's Bluetooth daemon or service.
// Upon complete initialization, it returns the session descriptor, and capabilities of
// the application.
func (b *BluetoothLibrary) Start(authHandler bluetooth.SessionAuthorizer, cfg config.Configuration) (*ac.FeatureSet, platforminfo.PlatformInfo, error) {
	var ce ac.Errors

	platform := platforminfo.NewPlatformInfo("Generic", implementation)

	var initialized bool
	defer func() {
		if !initialized {
			b.Stop()
		}
	}()

	b.Lock()
	defer b.Unlock()

	if authHandler == nil {
		authHandler = bluetooth.DefaultAuthorizer{}
	}

	b.authorizer = authHandler
	if err := lib.Initialize(authHandler, cfg); err != nil {
		return nil, platform, fault.Wrap(
			err,
			fctx.With(context.Background(), "error_at", "init-lib"),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot initialize the library"),
		)
	}

	b.store = sstore.NewSessionStore()
	if err := b.refreshStore(); err != nil {
		return nil, platform, fault.Wrap(
			err,
			fctx.With(context.Background(), "error_at", "init-session-store"),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot initialize the new session store"),
		)
	}

	features := lib.GetFeatures()
	for _, absentFeatures := range features.AbsentFeatures() {
		ce.Append(ac.NewError(absentFeatures, errorkinds.ErrNotSupported))
	}

	b.features = ac.NewFeatureSet(features, ce)
	if b.features.Has(ac.FeatureSendFile, ac.FeatureReceiveFile) {
		// TODO: Setup OPP server
		_ = 0
	}

	initialized = true
	b.sessionClosed.Store(false)

	return b.features, platform, nil
}

// Stop attempts to stop a session with the system's Bluetooth daemon or service.
func (b *BluetoothLibrary) Stop() error {
	b.Lock()
	defer b.Unlock()

	b.features = nil
	b.sessionClosed.Store(true)

	// TODO:Stop OPP server if started
	lib.Release()

	return nil
}

// Adapters returns a list of known adapters.
func (b *BluetoothLibrary) Adapters() ([]bluetooth.AdapterData, error) {
	return b.store.Adapters()
}

// Adapter returns a function call interface to invoke adapter related functions.
func (b *BluetoothLibrary) Adapter(address bluetooth.AdapterAddress) bluetooth.Adapter {
	return &adapter{s: b, key: address}
}

// Device returns a function call interface to invoke device related functions.
func (b *BluetoothLibrary) Device(address bluetooth.DeviceAddress) bluetooth.Device {
	return &device{s: b, key: address}
}

// Obex returns a function call interface to invoke obex related functions.
func (b *BluetoothLibrary) Obex(address bluetooth.DeviceAddress) bluetooth.Obex {
	return &obex{s: b, key: address}
}

// Network returns a function call interface to invoke network related functions.
func (b *BluetoothLibrary) Network(_ bluetooth.DeviceAddress) bluetooth.Network {
	return &network{}
}

// MediaPlayer returns a function call interface to invoke media player/control
// related functions on a device.
func (b *BluetoothLibrary) MediaPlayer(_ bluetooth.DeviceAddress) bluetooth.MediaPlayer {
	return &mediaPlayer{}
}

func (b *BluetoothLibrary) refreshStore() error {
	adapters, err := lib.GetAdapters()
	if err != nil {
		return err
	}

	for _, adapter := range adapters {
		b.store.AddAdapter(adapter)

		devices, err := lib.AdapterGetPairedDevices(adapter.AdapterAddress)
		if err != nil {
			return err
		}

		for _, device := range devices {
			b.store.AddDevice(device)
		}
	}

	return nil
}
