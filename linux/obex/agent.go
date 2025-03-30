//go:build linux

package obex

import (
	"errors"
	"path/filepath"
	"time"

	bluetooth "github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	dbh "github.com/bluetuith-org/bluetooth-classic/linux/internal/dbushelper"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

// agent describes an OBEX agent connection.
// Note that, all public methods are exported to the Obex Agent Manager
// via the session bus, and hence is called by the Agent Manager only.
// Any errors are published to the global error event stream.
type agent struct {
	authHandler bluetooth.AuthorizeReceiveFile

	ctx         bluetooth.AuthTimeout
	authTimeout time.Duration

	initialized bool

	fileTransfer
}

var obexAgent agent

// AuthorizePush asks for confirmation before receiving a transfer from the host device.
func (o *agent) AuthorizePush(transferPath dbus.ObjectPath) (string, *dbus.Error) {
	if !o.initialized {
		return "", nil
	}

	sessionPath := dbus.ObjectPath(filepath.Dir(string(transferPath)))

	sessionProperty, err := o.sessionProperties(sessionPath)
	if err != nil {
		dbh.PublishError(err,
			"OBEX agent error: Could not get session properties",
			"error_at", "authpush-session-properties",
		)

		return "", dbus.MakeFailedError(err)
	}

	transferProperty, err := o.transferProperties(transferPath)
	if err != nil {
		dbh.PublishError(err,
			"OBEX agent error: Could not get transfer properties",
			"error_at", "authpush-transfer-properties",
		)

		return "", dbus.MakeFailedError(err)
	}

	if sessionProperty.Root == "" {
		dbh.PublishError(err,
			"OBEX agent error: Session properties are empty",
			"error_at", "authpush-session-rootdest",
		)

		return "", dbus.MakeFailedError(errors.New("session property empty"))
	}

	if transferProperty.Status == bluetooth.TransferError {
		dbh.PublishError(err,
			"OBEX agent error: Transfer property is empty",
			"error_at", "authpush-transfer-status",
		)

		return "", dbus.MakeFailedError(errors.New("transfer property empty"))
	}

	transferProperty.Address = sessionProperty.Destination

	path := filepath.Join(sessionProperty.Root, transferProperty.Name)
	o.ctx = bluetooth.NewAuthTimeout(o.authTimeout)
	defer o.Cancel()

	if err := o.authHandler.AuthorizeTransfer(o.ctx, transferProperty); err != nil {
		dbh.PublishError(err,
			"OBEX agent error: Transfer was not authorized",
			"error_at", "authpush-agent-authorize",
		)

		return "", dbus.MakeFailedError(err)
	}

	return path, nil
}

// Cancel is called when the OBEX agent request was cancelled.
func (o *agent) Cancel() *dbus.Error {
	o.Cancel()

	return nil
}

// Release is called when the OBEX agent is unregistered.
func (o *agent) Release() *dbus.Error {
	return nil
}

// setupAgent sets up an OBEX agent.
func setupAgent(sessionBus *dbus.Conn, authHandler bluetooth.AuthorizeReceiveFile, authTimeout time.Duration) error {
	if authHandler == nil {
		return errors.New("No authorization handler interface specified")
	}

	ag := agent{authHandler: authHandler}
	ag.SessionBus = sessionBus

	err := sessionBus.Export(ag, dbh.ObexAgentPath, dbh.ObexAgentIface)
	if err != nil {
		return err
	}

	node := &introspect.Node{
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name:    dbh.ObexAgentIface,
				Methods: introspect.Methods(ag),
			},
		},
	}

	if err := sessionBus.Export(
		introspect.NewIntrospectable(node),
		dbh.ObexAgentPath,
		dbh.DbusIntrospectableIface,
	); err != nil {
		return err
	}

	if err := ag.callObexAgentManager("RegisterAgent", dbh.ObexAgentPath).Store(); err != nil {
		return err
	}

	ag.authTimeout = authTimeout

	obexAgent = ag

	return nil
}

// removeAgent removes the OBEX agent.
func removeAgent() error {
	if !obexAgent.initialized {
		return nil
	}

	return obexAgent.callObexAgentManager("UnregisterAgent", dbh.ObexAgentPath).Store()
}

// callObexAgentManager calls the OBEX AgentManager1 interface with the provided arguments.
func (o *agent) callObexAgentManager(method string, args ...interface{}) *dbus.Call {
	return o.SessionBus.Object(dbh.ObexBusName, dbh.ObexAgentManagerPath).
		Call(dbh.ObexAgentManagerIface+"."+method, 0, args...)
}
