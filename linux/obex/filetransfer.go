//go:build linux

package obex

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	bluetooth "github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	errorkinds "github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	dbh "github.com/bluetuith-org/bluetooth-classic/linux/internal/dbushelper"
	"github.com/godbus/dbus/v5"
)

// fileTransfer describes a file transfer session.
type fileTransfer Obex

// obexSessionProperties holds properties for a created Obex session.
type obexSessionProperties struct {
	Root        string
	Target      string
	Source      string
	Destination bluetooth.MacAddress
}

// CreateSession creates a new Obex session with a device.
// The context (ctx) can be provided in case this function call
// needs to be cancelled, since this function call can take some time
// to complete.
func (o *fileTransfer) CreateSession(ctx context.Context) error {
	if err := o.check(); err != nil {
		return err
	}

	var sessionPath dbus.ObjectPath

	args := make(map[string]interface{}, 1)
	args["Target"] = "opp"

	session := o.callClientAsync(ctx, "CreateSession", o.Address.String(), args)
	select {
	case <-ctx.Done():
		return fault.Wrap(
			context.Canceled,
			fctx.With(context.Background(),
				"error_at", "obex-createsession-cancelled",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Session creation was cancelled"),
		)

	case call := <-session.Done:
		if call.Err != nil {
			return fault.Wrap(
				call.Err,
				fctx.With(context.Background(),
					"error_at", "obex-createsession-methodcall",
					"address", o.Address.String(),
				),
				ftag.With(ftag.Internal),
				fmsg.With("Cannot start a file transfer session"),
			)
		}

		if err := call.Store(&sessionPath); err != nil {
			return fault.Wrap(
				err,
				fctx.With(context.Background(),
					"error_at", "obex-createsession-path",
					"address", o.Address.String(),
				),
				ftag.With(ftag.Internal),
				fmsg.With("Cannot obtain file transfer session data"),
			)
		}
	}

	dbh.PathConverter.AddDbusPath(dbh.DbusPathObexSession, sessionPath, o.Address)

	return nil
}

// RemoveSession removes a created Obex session.
func (o *fileTransfer) RemoveSession() error {
	if err := o.check(); err != nil {
		return err
	}

	sessionPath, ok := dbh.PathConverter.DbusPath(dbh.DbusPathObexSession, o.Address)
	if !ok {
		return fault.Wrap(
			errorkinds.ErrPropertyDataParse,
			fctx.With(context.Background(),
				"error_at", "obex-removesession-path",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot obtain file transfer session data"),
		)
	}

	if err := o.callClient("RemoveSession", sessionPath).Store(); err != nil {
		return fault.Wrap(
			err,
			fctx.With(context.Background(),
				"error_at", "obex-removesession-methodcall",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("An error occurred while removing the file transfer session"),
		)
	}

	return nil
}

// SendFile sends a file to the device. The 'filepath' must be a full path to the file.
func (o *fileTransfer) SendFile(filepath string) (bluetooth.FileTransferData, error) {
	if err := o.check(); err != nil {
		return bluetooth.FileTransferData{}, err
	}

	var transferPath dbus.ObjectPath

	var fileTransferObject bluetooth.FileTransferData

	sessionPath, ok := dbh.PathConverter.DbusPath(dbh.DbusPathObexSession, o.Address)
	if !ok {
		return bluetooth.FileTransferData{},
			fault.Wrap(
				errorkinds.ErrPropertyDataParse,
				fctx.With(context.Background(),
					"error_at", "obex-sendfile-sessionpath",
					"address", o.Address.String(),
				),
				ftag.With(ftag.Internal),
				fmsg.With("Cannot obtain file transfer session data"),
			)
	}

	transferPropertyMap := make(map[string]dbus.Variant)
	if err := o.callObjectPush(sessionPath, "SendFile", filepath).
		Store(&transferPath, &transferPropertyMap); err != nil {
		return bluetooth.FileTransferData{},
			fault.Wrap(
				err,
				fctx.With(context.Background(),
					"error_at", "obex-sendfile-methodcall",
					"address", o.Address.String(),
				),
				ftag.With(ftag.Internal),
				fmsg.With("Cannot send file: "+filepath),
			)
	}

	dbh.PathConverter.AddDbusPath(dbh.DbusPathObexTransfer, transferPath, o.Address)

	if err := dbh.DecodeVariantMap(transferPropertyMap, &fileTransferObject); err != nil {
		return bluetooth.FileTransferData{},
			fault.Wrap(
				err,
				fctx.With(context.Background(),
					"error_at", "obex-sendfile-decode",
					"address", o.Address.String(),
				),
				ftag.With(ftag.Internal),
				fmsg.With("Cannot obtain file transfer data"),
			)
	}

	return fileTransferObject, nil
}

// CancelTransfer cancels the transfer.
func (o *fileTransfer) CancelTransfer() error {
	if err := o.check(); err != nil {
		return err
	}

	transferPath, ok := dbh.PathConverter.DbusPath(dbh.DbusPathObexTransfer, o.Address)
	if !ok {
		return fault.Wrap(
			errorkinds.ErrPropertyDataParse,
			fctx.With(context.Background(),
				"error_at", "obex-canceltransfer-path",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot obtain file transfer data"),
		)
	}

	if err := o.callTransfer(transferPath, "Cancel").Store(); err != nil {
		return fault.Wrap(
			err,
			fctx.With(context.Background(),
				"error_at", "obex-canceltransfer-call",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot cancel transfer"),
		)
	}

	return nil
}

// SuspendTransfer suspends the transfer.
func (o *fileTransfer) SuspendTransfer() error {
	if err := o.check(); err != nil {
		return err
	}

	transferPath, ok := dbh.PathConverter.DbusPath(dbh.DbusPathObexTransfer, o.Address)
	if !ok {
		return fault.Wrap(
			errorkinds.ErrPropertyDataParse,
			fctx.With(context.Background(),
				"error_at", "obex-suspendtransfer-path",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot obtain file transfer data"),
		)
	}

	if err := o.callTransfer(transferPath, "Suspend").Store(); err != nil {
		return fault.Wrap(
			err,
			fctx.With(context.Background(),
				"error_at", "obex-suspendtransfer-call",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot suspend transfer"),
		)
	}

	return nil
}

// ResumeTransfer resumes the transfer.
func (o *fileTransfer) ResumeTransfer() error {
	if err := o.check(); err != nil {
		return err
	}

	transferPath, ok := dbh.PathConverter.DbusPath(dbh.DbusPathObexTransfer, o.Address)
	if !ok {
		return fault.Wrap(
			errorkinds.ErrPropertyDataParse,
			fctx.With(context.Background(),
				"error_at", "obex-resumetransfer-path",
				"address", o.Address.String(),
			),
			ftag.With(ftag.NotFound),
			fmsg.With("Cannot obtain file transfer data"),
		)
	}

	if err := o.callTransfer(transferPath, "Resume").Store(); err != nil {
		return fault.Wrap(
			err,
			fctx.With(context.Background(),
				"error_at", "obex-resumetransfer-call",
				"address", o.Address.String(),
			),
			ftag.With(ftag.Internal),
			fmsg.With("Cannot resume transfer"),
		)
	}

	return nil
}

// check checks whether the SessionBus was initialized.
func (o *fileTransfer) check() error {
	if o.SessionBus == nil {
		return fault.Wrap(errorkinds.ErrObexInitSession,
			fctx.With(context.Background(),
				"error_at", "obex-check-sessionbus",
				"address", o.Address.String(),
			),
			ftag.With(ftag.NotFound),
			fmsg.With("Cannot call file transfer method on session-bus"),
		)
	}

	_, ok := dbh.PathConverter.DbusPath(dbh.DbusPathDevice, o.Address)
	if !ok {
		return fault.Wrap(errorkinds.ErrDeviceNotFound,
			fctx.With(context.Background(),
				"error_at", "obex-check-device",
				"address", o.Address.String(),
			),
			ftag.With(ftag.NotFound),
			fmsg.With("Device does not exist"),
		)
	}

	return nil
}

// callClient calls the Client1 interface with the provided method.
func (o *fileTransfer) callClient(method string, args ...interface{}) *dbus.Call {
	return o.SessionBus.Object(dbh.ObexBusName, dbh.ObexBusPath).
		Call(dbh.ObexClientIface+"."+method, 0, args...)
}

// callClientAsync calls the Client1 interface asynchronously with the provided method.
func (o *fileTransfer) callClientAsync(ctx context.Context, method string, args ...interface{}) *dbus.Call {
	return o.SessionBus.Object(dbh.ObexBusName, dbh.ObexBusPath).
		GoWithContext(ctx, dbh.ObexClientIface+"."+method, 0, nil, args...)
}

// callObjectPush calls the ObjectPush1 interface with the provided method.
func (o *fileTransfer) callObjectPush(sessionPath dbus.ObjectPath, method string, args ...interface{}) *dbus.Call {
	return o.SessionBus.Object(dbh.ObexBusName, sessionPath).
		Call(dbh.ObexObjectPushIface+"."+method, 0, args...)
}

// callTransfer calls the Transfer1 interface with the provided method.
func (o *fileTransfer) callTransfer(transferPath dbus.ObjectPath, method string, args ...interface{}) *dbus.Call {
	return o.SessionBus.Object(dbh.ObexBusName, transferPath).
		Call(dbh.ObexTransferIface+"."+method, 0, args...)
}

// sessionProperties converts a map of OBEX session properties to ObexSessionProperties.
func (o *fileTransfer) sessionProperties(sessionPath dbus.ObjectPath) (obexSessionProperties, error) {
	var sessionProperties obexSessionProperties

	props := make(map[string]dbus.Variant)
	if err := o.SessionBus.Object(dbh.ObexBusName, sessionPath).
		Call(dbh.DbusGetAllPropertiesIface, 0, dbh.ObexSessionIface).
		Store(&props); err != nil {
		return obexSessionProperties{}, err
	}

	return sessionProperties, dbh.DecodeVariantMap(props, &sessionProperties)
}

// transferProperties converts a map of OBEX transfer properties to FileTransferData.
func (o *fileTransfer) transferProperties(transferPath dbus.ObjectPath) (bluetooth.FileTransferData, error) {
	var transferProperties bluetooth.FileTransferData

	props := make(map[string]dbus.Variant)
	if err := o.SessionBus.Object(dbh.ObexBusName, transferPath).
		Call(dbh.DbusGetAllPropertiesIface, 0, dbh.ObexTransferIface).
		Store(&props); err != nil {
		return bluetooth.FileTransferData{}, err
	}

	return transferProperties, dbh.DecodeVariantMap(props, &transferProperties)
}
