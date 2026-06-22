//go:build !linux && libhbluetooth

package lib

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	ffi "github.com/bluetuith-org/libffi-go"
)

type oppTransferStatus uint32

const (
	oppStatusNone oppTransferStatus = iota
	oppStatusQueued
	oppStatusActive
	oppStatusSuspended
	pppStatusComplete
	oppStatusError
)

type oppTransferData struct {
	ID deviceIDNative

	Name     *byte
	FileName *byte

	Receiving bool
	FileSize  int64

	Status           oppTransferStatus
	BytesTransferred int64
	SessionID        uint64
	TransferID       uint64
}

func (o *oppTransferData) toObjectPushData() bluetooth.ObjectPushData {
	getStatus := func(status oppTransferStatus) bluetooth.ObjectPushStatus {
		switch status {
		case oppStatusNone:
			return ""

		case oppStatusQueued:
			return bluetooth.TransferQueued

		case oppStatusActive:
			return bluetooth.TransferActive

		case oppStatusSuspended:
			return bluetooth.TransferSuspended

		case pppStatusComplete:
			return bluetooth.TransferComplete

		case oppStatusError:
			return bluetooth.TransferError

		default:
			return ""
		}
	}

	getSessionID := func(sessionID uint64) bluetooth.ObjectPushSessionID {
		var sb strings.Builder

		fmt.Fprintf(&sb, "sid%d", sessionID)

		return bluetooth.ObjectPushSessionID(sb.String())
	}

	getTransferID := func(sessionID, transferID uint64) bluetooth.ObjectPushTransferID {
		var sb strings.Builder

		fmt.Fprintf(&sb, "sid%d/tid%d", sessionID, transferID)

		return bluetooth.ObjectPushTransferID(sb.String())
	}

	data := bluetooth.ObjectPushData{
		Name:      bytePtrToString(o.Name),
		Filename:  bytePtrToString(o.FileName),
		Receiving: o.Receiving,
		ObjectPushEventData: bluetooth.ObjectPushEventData{
			DeviceAddress: o.ID.ToDeviceAddress(),
			Status:        getStatus(o.Status),
			Size:          uint64(o.FileSize),
			Transferred:   uint64(o.BytesTransferred),
			SessionID:     getSessionID(o.SessionID),
			TransferID:    getTransferID(o.SessionID, o.TransferID),
		},
	}

	data.Type = mime.TypeByExtension(filepath.Ext(data.Filename))

	return data
}

// OppCreateSession opens an Object Push transfer session with the target device.
func OppCreateSession(deviceAddress bluetooth.DeviceAddress) error {
	return oppCallDeviceFunc(deviceAddress, _hbcOppStartSession)
}

// OppRemoveSession closes an Object Push transfer session with the target device.
func OppRemoveSession(deviceAddress bluetooth.DeviceAddress) error {
	return oppCallDeviceFunc(deviceAddress, _hbcOppStopSession)
}

// OppQueueFileToSend queues a file to send to the target device. Ensure [lib.OppCreateSession] is called before using this function.
func OppQueueFileToSend(deviceAddress bluetooth.DeviceAddress, file string) (bluetooth.ObjectPushData, error) {
	libErr := newLibError()

	argDeviceID := newDeviceID(deviceAddress)
	argFilePath := stringToBytePtr(file)
	argOppData := &oppTransferData{}

	_hbcOppQueueFile.Call(libErr.getReturnPtr(), &argDeviceID, &argFilePath, &argOppData, libErr.getHbErrorPtr())
	if err := libErr.getError(); err != nil {
		return bluetooth.ObjectPushData{}, err
	}

	return argOppData.toObjectPushData(), nil
}

// OppCancelTransfer cancels a transfer.
func OppCancelTransfer(deviceAddress bluetooth.DeviceAddress) error {
	return oppCallDeviceFunc(deviceAddress, _hbcOppCancelTransfer)
}

// OppSuspendTransfer suspends a transfer.
func OppSuspendTransfer(_ bluetooth.DeviceAddress) error {
	return errorkinds.ErrNotSupported
}

// OppResumeTransfer resumes a transfer.
func OppResumeTransfer(_ bluetooth.DeviceAddress) error {
	return errorkinds.ErrNotSupported
}

// OppStartServer starts the Object Push server, to receive Object Push transfers.
func OppStartServer(adapterAddress bluetooth.AdapterAddress) error {
	return oppCallAdapterFunc(adapterAddress, _hbcOppStartServer)
}

// OppStopServer stops the Object Push server.
func OppStopServer(adapterAddress bluetooth.AdapterAddress) error {
	return oppCallAdapterFunc(adapterAddress, _hbcOppStopServer)
}

func oppCallDeviceFunc(deviceAddress bluetooth.DeviceAddress, fn ffi.Fun) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(deviceAddress)

	fn.Call(libErr.getReturnPtr(), &argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError()
}

func oppCallAdapterFunc(adapterAddress bluetooth.AdapterAddress, fn ffi.Fun) error {
	libErr := newLibError()

	argAdapterAddress := newBdAddr(adapterAddress.Address)

	fn.Call(libErr.getReturnPtr(), &argAdapterAddress, libErr.getHbErrorPtr())

	return libErr.getError()
}

func handleOppEvent(action bluetooth.EventAction, data *oppTransferData) {
	oppData := data.toObjectPushData()

	switch action {
	case bluetooth.EventActionAdded:
		bluetooth.ObjectPushEvents().PublishAdded(oppData)

	case bluetooth.EventActionUpdated:
		bluetooth.ObjectPushEvents().PublishUpdated(oppData.ObjectPushEventData)

	case bluetooth.EventActionRemoved:
		bluetooth.ObjectPushEvents().PublishRemoved(oppData.ObjectPushEventData)
	}
}

var (
	_hbcOppStartSession, _hbcOppStopSession ffi.Fun

	_hbcOppQueueFile      ffi.Fun
	_hbcOppCancelTransfer ffi.Fun

	_hbcOppStartServer, _hbcOppStopServer ffi.Fun
)

func getOppFunHandles() []funHandle {
	return []funHandle{
		{
			&_hbcOppStartSession, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_opp_start_session", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcOppStopSession, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_opp_stop_session", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcOppQueueFile, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_opp_queue_file_path", &fnRetType, &ffi.TypePointer, &ffi.TypePointer, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcOppCancelTransfer, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_opp_cancel_transfer", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcOppStartServer, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_opp_start_server", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbcOppStopServer, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hbc_opp_stop_server", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
	}
}
