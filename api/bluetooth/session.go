package bluetooth

import (
	ac "github.com/bluetuith-org/bluetooth-classic/api/appfeatures"
	"github.com/bluetuith-org/bluetooth-classic/api/config"
	"github.com/bluetuith-org/bluetooth-classic/api/platforminfo"
)

// Session describes a Bluetooth application session.
type Session interface {
	// Start attempts to initialize a session with the system's Bluetooth daemon or service.
	// Upon complete initialization, it returns the session descriptor, and capabilities of
	// the application.
	Start(authHandler SessionAuthorizer, cfg config.Configuration) (*ac.FeatureSet, platforminfo.PlatformInfo, error)

	// Stop attempts to stop a session with the system's Bluetooth daemon or service.
	Stop() error

	// Adapters returns a list of known adapters.
	Adapters() ([]AdapterData, error)

	// Adapter returns a function call interface to invoke adapter related functions.
	Adapter(address AdapterAddress) Adapter

	// Device returns a function call interface to invoke device related functions.
	Device(address DeviceAddress) Device

	// Obex returns a function call interface to invoke obex related functions.
	Obex(address DeviceAddress) Obex

	// Network returns a function call interface to invoke network related functions.
	Network(address DeviceAddress) Network

	// MediaPlayer returns a function call interface to invoke media player/control
	// related functions on a device.
	MediaPlayer(address DeviceAddress) MediaPlayer
}
