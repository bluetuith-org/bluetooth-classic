//go:build linux

package bluez

import (
	"context"
	"maps"
	"path/filepath"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	ac "github.com/bluetuith-org/bluetooth-classic/api/appfeatures"
	bluetooth "github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/config"
	errorkinds "github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/bluetuith-org/bluetooth-classic/api/helpers/sessionstore"
	"github.com/bluetuith-org/bluetooth-classic/api/platforminfo"
	dbh "github.com/bluetuith-org/bluetooth-classic/internal/bluez/internal/dbushelper"
	mp "github.com/bluetuith-org/bluetooth-classic/internal/bluez/mediaplayer"
	nm "github.com/bluetuith-org/bluetooth-classic/internal/bluez/networkmanager"
	"github.com/bluetuith-org/bluetooth-classic/internal/bluez/obex"
	"github.com/godbus/dbus/v5"
)

const implementation = "BlueZ"

// DbusSession describes a Linux Bluez DBus session.
type DbusSession struct {
	systemBus  *dbus.Conn
	sessionBus *dbus.Conn
	agent      *agent

	netman  *nm.NetManager
	obexman *obex.ObexManager

	store sessionstore.SessionStore
}

// Start attempts to initialize and start interfacing with the Bluez daemon via DBus.
func (b *DbusSession) Start(
	authHandler bluetooth.SessionAuthorizer,
	cfg config.Configuration,
) (*ac.FeatureSet, platforminfo.PlatformInfo, error) {
	var capabilities ac.Features
	var ce ac.Errors

	if authHandler == nil {
		authHandler = &bluetooth.DefaultAuthorizer{}
	}

	platform := platforminfo.NewPlatformInfo("BlueZ (DBus)", implementation)

	systemBus, err := dbus.SystemBus()
	if err != nil {
		return nil, platform,
			fault.Wrap(
				err,
				fctx.With(context.Background(), "error_at", "start-systembus"),
				ftag.With(ftag.Internal),
				fmsg.With("Cannot initialize system DBus"),
			)
	}

	var sessionBus *dbus.Conn
	if cfg.EnableObexServices {
		sessionBus, err = dbus.SessionBus()
		if err != nil {
			return nil, platform,
				fault.Wrap(
					err,
					fctx.With(context.Background(), "error_at", "start-sessionbus"),
					ftag.With(ftag.Internal),
					fmsg.With("Cannot start session DBus"),
				)
		}
	}

	*b = DbusSession{
		systemBus:  systemBus,
		sessionBus: sessionBus,
		store:      sessionstore.NewSessionStore(),
	}

	if err := b.refreshStore(); err != nil {
		return nil, platform,
			fault.Wrap(
				err,
				fctx.With(context.Background(), "error_at", "refresh-sessionstore"),
				ftag.With(ftag.Internal),
				fmsg.With("Error while initializing object cache"),
			)
	}

	b.agent = newAgent(systemBus, authHandler, cfg.AuthTimeout)
	if err := b.agent.setup(); err != nil {
		return nil, platform,
			fault.Wrap(
				err,
				fctx.With(context.Background(), "error_at", "agent-initialize"),
				ftag.With(ftag.Internal),
				fmsg.With("Error while initializing Bluez agent"),
			)
	}

	capabilities.Add(
		ac.FeatureConnection,
		ac.FeaturePairing,
		ac.FeatureMediaPlayer,
	)

	b.obexman = obex.NewManager(sessionBus)
	obexcap, cerr := b.obexman.Initialize(authHandler, cfg.AuthTimeout)
	if cerr != nil {
		ce.Append(cerr)
	}

	netman, netcap, cerr := nm.Initialize()
	if cerr != nil {
		ce.Append(cerr)
	} else {
		b.netman = netman
	}

	capabilities.Add(obexcap, netcap)

	go b.watchBluezSystemBus()

	return ac.NewFeatureSet(capabilities, ce), platform, nil
}

// Stop attempts to stop interfacing with the Bluez daemon.
func (b *DbusSession) Stop() error {
	_ = b.obexman.Stop()
	_ = b.agent.remove()

	if b.sessionBus != nil {
		if err := b.sessionBus.Close(); err != nil {
			return fault.Wrap(
				err,
				fctx.With(context.Background(), "error_at", "stop-sessionbus"),
				ftag.With(ftag.Internal),
				fmsg.With("Error while closing session bus"),
			)
		}
	}

	if err := b.systemBus.Close(); err != nil {
		return fault.Wrap(
			err,
			fctx.With(context.Background(), "error_at", "stop-systembus"),
			ftag.With(ftag.Internal),
			fmsg.With("Error while closing system bus"),
		)
	}

	return nil
}

// Adapters returns a list of known adapters.
func (b *DbusSession) Adapters() ([]bluetooth.AdapterData, error) {
	return b.store.Adapters()
}

// Adapter returns a function call interface to invoke adapter related functions.
func (b *DbusSession) Adapter(address bluetooth.AdapterAddress) bluetooth.Adapter {
	return &adapter{b: b, key: address}
}

// Device returns a function call interface to invoke device related functions.
func (b *DbusSession) Device(address bluetooth.DeviceAddress) bluetooth.Device {
	return &device{b: b, key: address}
}

// Obex returns a function call interface to invoke obex related functions.
func (b *DbusSession) Obex(address bluetooth.DeviceAddress) bluetooth.Obex {
	return &obex.Obex{SessionBus: b.sessionBus, Key: address}
}

// Network returns a function call interface to invoke network related functions.
func (b *DbusSession) Network(address bluetooth.DeviceAddress) bluetooth.Network {
	return &nm.Network{NetManager: b.netman, Key: address}
}

// MediaPlayer returns a function call interface to invoke mediaplayer related functions.
func (b *DbusSession) MediaPlayer(address bluetooth.DeviceAddress) bluetooth.MediaPlayer {
	return &mp.MediaPlayer{SystemBus: b.systemBus, Key: address}
}

// adapterInternal returns an adapter-related function call interface for internal use.
// This is used primarily to initialize adapterInternal objects.
func (b *DbusSession) adapterInternal(path dbus.ObjectPath) *adapter {
	return &adapter{b: b, path: path}
}

// deviceInternal returns an device-related function call interface for internal use.
// This is used primarily to initialize deviceInternal objects.
func (b *DbusSession) deviceInternal(path dbus.ObjectPath) *device {
	return &device{b: b, path: path}
}

// mediaPlayerInternal returns an mediaplayer-related function call interface for internal use.
// This is used primarily to initialize mediaPlayerInternal objects.
func (b *DbusSession) mediaPlayerInternal() *mp.MediaPlayer {
	return &mp.MediaPlayer{SystemBus: b.systemBus}
}

// refreshStore refreshes the global session store with adapter and device objects
// that are retrieved from the Bluez DBus interface (system bus).
func (b *DbusSession) refreshStore() error {
	objects := make(map[dbus.ObjectPath]map[string]map[string]dbus.Variant)
	if err := b.systemBus.Object(dbh.BluezBusName, "/").
		Call(dbh.DbusObjectManagerIface, 0).
		Store(&objects); err != nil {
		return err
	}

	for path, object := range objects {
		for iface, values := range object {
			var err error

			switch iface {
			case dbh.BluezAdapterIface:
				_, err = b.adapterInternal(path).convertAndStoreObjects(values)

			case dbh.BluezDeviceIface:
				_, err = b.deviceInternal(path).convertAndStoreObjects(values)
			}

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// watchBluezSystemBus will register a signal to receive events from the bluez dbus interface.
func (b *DbusSession) watchBluezSystemBus() {
	signalMatch := "type='signal', sender='org.bluez'"
	b.systemBus.BusObject().Call(dbh.DbusSignalAddMatchIface, 0, signalMatch)

	ch := make(chan *dbus.Signal, 1)
	b.systemBus.Signal(ch)

	for signal := range ch {
		b.parseSignalData(signal)
	}
}

// parseSignalData parses bluez DBus signal data.
//
//gocyclo:ignore
func (b *DbusSession) parseSignalData(signal *dbus.Signal) {
	switch signal.Name {
	case dbh.DbusSignalPropertyChangedIface:
		if signal.Body != nil && len(signal.Body) < 2 {
			return
		}

		objectInterfaceName, ok := signal.Body[0].(string)
		if !ok {
			return
		}

		propertyMap, ok := signal.Body[1].(map[string]dbus.Variant)
		if !ok {
			return
		}

		switch objectInterfaceName {
		case dbh.BluezAdapterIface:
			dbh.PublishAdapterUpdateEvent(&b.store, signal, propertyMap)

		case dbh.BluezDeviceIface:
			dbh.PublishDeviceUpdateEvent(&b.store, signal, propertyMap)

		case dbh.BluezMediaPlayerIface:
			devicePath := dbus.ObjectPath(filepath.Dir(string(signal.Path)))

			key, ok := dbh.PathConverter.DeviceAddress(dbh.DbusPathDevice, devicePath)
			if !ok {
				dbh.PublishSignalError(
					errorkinds.ErrDeviceNotFound, signal,
					"Bluez event handler error",
					"error_at", "pchanged-mediaplayer-address",
				)

				return
			}

			properties, err := b.mediaPlayerInternal().ParseMap(propertyMap)
			if err != nil {
				dbh.PublishSignalError(
					err, signal,
					"Bluez event handler error",
					"error_at", "pchanged-mediaplayer-address",
				)

				return
			}

			properties.DeviceAddress = key

			bluetooth.MediaEvents().PublishUpdated(properties)

		case dbh.BluezBatteryIface:
			percentage := -1

			if v, ok := propertyMap["Percentage"]; ok {
				if p, ok := v.Value().(byte); ok {
					percentage = int(p)
				}
			}

			if percentage < 0 {
				dbh.PublishSignalError(
					errorkinds.ErrEventDataParse, signal,
					"Bluez event handler error",
					"error_at", "pchanged-batterypct-decode",
				)

				return
			}

			dbh.PublishDeviceUpdateEvent(&b.store, signal, propertyMap)
		}

	case dbh.DbusSignalInterfacesAddedIface:
		if signal.Body != nil && len(signal.Body) < 2 {
			return
		}

		objectPath, ok := signal.Body[0].(dbus.ObjectPath)
		if !ok {
			return
		}

		nestedPropertyMap, ok := signal.Body[1].(map[string]map[string]dbus.Variant)
		if !ok {
			return
		}

		for iftype := range nestedPropertyMap {
			mergedPropertyMap, ok := nestedPropertyMap[iftype]
			if !ok {
				return
			}

			for key, values := range nestedPropertyMap {
				if key == iftype {
					continue
				}

				maps.Copy(mergedPropertyMap, values)
			}

			switch iftype {
			case dbh.BluezAdapterIface:
				adapter, err := b.adapterInternal(objectPath).convertAndStoreObjects(mergedPropertyMap)
				if err != nil {
					dbh.PublishSignalError(
						err, signal,
						"Bluez event handler error",
						"error_at", "padded-adapter-decode",
					)

					continue
				}

				b.store.AddAdapter(adapter)
				dbh.PathConverter.AddAdapterDbusPath(objectPath, adapter.AdapterAddress)

				bluetooth.AdapterEvents().PublishAdded(adapter)

			case dbh.BluezDeviceIface:
				device, err := b.deviceInternal(objectPath).convertAndStoreObjects(mergedPropertyMap)
				if err != nil {
					dbh.PublishSignalError(
						err, signal,
						"Bluez event handler error",
						"error_at", "padded-device-decode",
					)

					continue
				}

				b.store.AddDevice(device)
				dbh.PathConverter.AddDeviceDbusPath(
					dbh.DbusPathDevice,
					objectPath,
					device.DeviceAddress,
				)

				bluetooth.DeviceEvents().PublishAdded(device)

			case dbh.BluezBatteryIface:
				percentage := -1

				propertyMap := nestedPropertyMap[iftype]

				if v, ok := propertyMap["Percentage"]; ok {
					if p, ok := v.Value().(byte); ok {
						percentage = int(p)
					}
				}

				if percentage < 0 {
					dbh.PublishSignalError(
						errorkinds.ErrEventDataParse, signal,
						"Bluez event handler error",
						"error_at", "padded-batterypct-decode",
					)

					return
				}

				signal.Path = objectPath
				dbh.PublishDeviceUpdateEvent(&b.store, signal, propertyMap)
			}
		}

	case dbh.DbusSignalInterfacesRemovedIface:
		if signal.Body != nil && len(signal.Body) < 2 {
			return
		}

		objectPath, ok := signal.Body[0].(dbus.ObjectPath)
		if !ok {
			return
		}

		ifaceNames, ok := signal.Body[1].([]string)
		if !ok {
			return
		}

		for _, ifaceName := range ifaceNames {
			switch ifaceName {
			case dbh.BluezAdapterIface:
				key, ok := dbh.PathConverter.AdapterAddress(objectPath)
				if !ok {
					dbh.PublishSignalError(
						errorkinds.ErrAdapterNotFound, signal,
						"Bluez event handler error",
						"error_at", "premoved-adapter-address",
					)

					return
				}

				adapter := bluetooth.AdapterEventData{AdapterAddress: key}
				b.store.RemoveAdapter(key)
				dbh.PathConverter.RemoveAdapterDbusPath(objectPath)

				bluetooth.AdapterEvents().PublishRemoved(adapter)

			case dbh.BluezDeviceIface:
				key, ok := dbh.PathConverter.DeviceAddress(dbh.DbusPathDevice, objectPath)
				if !ok {
					dbh.PublishSignalError(
						errorkinds.ErrDeviceNotFound, signal,
						"Bluez event handler error",
						"error_at", "premoved-device-address",
					)

					return
				}

				device := bluetooth.DeviceEventData{
					DeviceAddress: key,
				}

				b.store.RemoveDevice(device.DeviceAddress)
				dbh.PathConverter.RemoveDeviceDbusPath(dbh.DbusPathDevice, objectPath)

				bluetooth.DeviceEvents().PublishRemoved(device)
			}
		}
	}
}
