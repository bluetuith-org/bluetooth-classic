//go:build linux

package obex

import (
	"errors"
	"path/filepath"
	"time"

	ac "github.com/bluetuith-org/bluetooth-classic/api/appfeatures"
	bluetooth "github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	errorkinds "github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	dbh "github.com/bluetuith-org/bluetooth-classic/linux/internal/dbushelper"
	"github.com/godbus/dbus/v5"
)

// Obex describes a Bluez Obex session.
type Obex struct {
	SessionBus *dbus.Conn
	Address    bluetooth.MacAddress
}

// ObexManager holds an OBEX session and agent.
type ObexManager struct {
	agent      *agent
	sessionBus *dbus.Conn
}

// NewManager returns a new ObexManager.
func NewManager(sessionBus *dbus.Conn) *ObexManager {
	return &ObexManager{
		sessionBus: sessionBus,
	}
}

// Initialize attempts to initialize the Obex Agent, and returns the capabilities of the
// obex session.
func (o *ObexManager) Initialize(auth bluetooth.AuthorizeReceiveFile, authTimeout time.Duration) (ac.Features, *ac.Error) {
	var capabilities ac.Features

	serviceNames, err := dbh.ListActivatableBusNames(o.sessionBus)
	if err != nil {
		return capabilities,
			ac.NewError(ac.FeatureSendFile|ac.FeatureReceiveFile, err)
	}

	for _, name := range serviceNames {
		if name == dbh.ObexBusName {
			goto SetupAgent
		}
	}

	return capabilities,
		ac.NewError(
			ac.FeatureSendFile|ac.FeatureReceiveFile,
			errors.New("OBEX Service does not exist"),
		)

SetupAgent:
	go o.watchObexSystemBus()

	capabilities = ac.FeatureSendFile

	o.agent = newAgent(auth, authTimeout, &fileTransfer{SessionBus: o.sessionBus})
	if err := o.agent.setup(); err != nil {
		return capabilities,
			ac.NewError(ac.FeatureReceiveFile, err)
	}

	capabilities |= ac.FeatureReceiveFile

	return capabilities, nil
}

// Stop removes the obex agent and closes the obex session.
func (o *ObexManager) Stop() error {
	return o.agent.remove()
}

// FileTransfer returns a function call interface to invoke device file transfer
// related functions.
func (o *Obex) FileTransfer() bluetooth.ObexFileTransfer {
	return &fileTransfer{SessionBus: o.SessionBus, Address: o.Address}
}

// watchObexSystemBus will register a signal and watch for events from the OBEX DBus interface.
func (o *ObexManager) watchObexSystemBus() {
	signalMatch := "type='signal', sender='org.bluez.obex'"
	o.sessionBus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, signalMatch)

	ch := make(chan *dbus.Signal, 1)
	o.sessionBus.Signal(ch)

	for signal := range ch {
		o.parseSignalData(signal)
	}
}

// parseSignalData parses OBEX DBus signal data.
func (o *ObexManager) parseSignalData(signal *dbus.Signal) {
	// BUG: Handle session and transfer interfaces when files are received.
	// BUG: dbh.DbusSignalPropertyAddedIface unhandled.
	switch signal.Name {
	case dbh.DbusSignalPropertyChangedIface:
		objectInterfaceName, ok := signal.Body[0].(string)
		if !ok {
			return
		}

		propertyMap, ok := signal.Body[1].(map[string]dbus.Variant)
		if !ok {
			return
		}

		switch objectInterfaceName {
		case dbh.ObexTransferIface:
			sessionPath := dbus.ObjectPath(filepath.Dir(string(signal.Path)))

			address, ok := dbh.PathConverter.Address(dbh.DbusPathObexSession, sessionPath)
			if !ok {
				dbh.PublishSignalError(errorkinds.ErrDeviceNotFound, signal,
					"Obex event handler error",
					"error_at", "pchanged-obex-address",
				)

				return
			}

			transferData := bluetooth.FileTransferEventData{
				Address: address,
			}
			if err := dbh.DecodeVariantMap(
				propertyMap, &transferData,
				"Status", "Transferred",
			); err != nil {
				dbh.PublishSignalError(err, signal,
					"Obex event handler error",
					"error_at", "pchanged-obex-decode",
				)

				return
			}

			bluetooth.FileTransferEvent(bluetooth.EventActionUpdated).PublishData(transferData)
		}

	case dbh.DbusSignalInterfacesRemovedIface:
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
			case dbh.ObexSessionIface:
				dbh.PathConverter.RemoveDbusPath(dbh.DbusPathObexSession, objectPath)

			case dbh.ObexTransferIface:
				dbh.PathConverter.RemoveDbusPath(dbh.DbusPathObexTransfer, objectPath)
			}
		}
	}
}
