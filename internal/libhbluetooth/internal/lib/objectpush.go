//go:build !linux && libhbluetooth

package lib

import (
	"fmt"
	"mime"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/ebitengine/purego"
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
	libErr := newLibError()

	argDeviceID := newDeviceID(deviceAddress)
	ret := _hbcOppStartSession.Call(argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// OppRemoveSession closes an Object Push transfer session with the target device.
func OppRemoveSession(deviceAddress bluetooth.DeviceAddress) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(deviceAddress)
	ret := _hbcOppStopSession.Call(argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// OppQueueFileToSend queues a file to send to the target device. Ensure [lib.OppCreateSession] is called before using this function.
func OppQueueFileToSend(deviceAddress bluetooth.DeviceAddress, file string) (bluetooth.ObjectPushData, error) {
	libErr := newLibError()

	argDeviceID := newDeviceID(deviceAddress)
	argFilePath := stringToBytePtr(file)
	argOppData := &oppTransferData{}

	ret := _hbcOppQueueFile.Call(argDeviceID, argFilePath, argOppData, libErr.getHbErrorPtr())
	if err := libErr.getError(ret); err != nil {
		return bluetooth.ObjectPushData{}, err
	}

	return argOppData.toObjectPushData(), nil
}

// OppCancelTransfer cancels a transfer.
func OppCancelTransfer(deviceAddress bluetooth.DeviceAddress) error {
	libErr := newLibError()

	argDeviceID := newDeviceID(deviceAddress)
	ret := _hbcOppCancelTransfer.Call(argDeviceID, libErr.getHbErrorPtr())

	return libErr.getError(ret)
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
	libErr := newLibError()

	argAdapterAddress := newBdAddr(adapterAddress.Address)
	ret := _hbcOppStartServer.Call(argAdapterAddress, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

// OppStopServer stops the Object Push server.
func OppStopServer(adapterAddress bluetooth.AdapterAddress) error {
	libErr := newLibError()

	argAdapterAddress := newBdAddr(adapterAddress.Address)
	ret := _hbcOppStopServer.Call(argAdapterAddress, libErr.getHbErrorPtr())

	return libErr.getError(ret)
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
	_hbcOppStartSession, _hbcOppStopSession interopFunc[func(*deviceIDNative, **hbError) hbStatus]

	_hbcOppQueueFile      interopFunc[func(*deviceIDNative, *byte, *oppTransferData, **hbError) hbStatus]
	_hbcOppCancelTransfer interopFunc[func(*deviceIDNative, **hbError) hbStatus]

	_hbcOppStartServer, _hbcOppStopServer interopFunc[func(*bdAddr, **hbError) hbStatus]
)

func getOppFunHandles() []funHandle {
	return []funHandle{
		newInteropFunc("hbc_opp_start_session", &_hbcOppStartSession, func(id *deviceIDNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcOppStartSession.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_opp_stop_session", &_hbcOppStopSession, func(id *deviceIDNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcOppStopSession.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_opp_queue_file_path", &_hbcOppQueueFile, func(id *deviceIDNative, path *byte, data *oppTransferData, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcOppQueueFile.funAddr(), uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(data)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(id)
			runtime.KeepAlive(path)
			runtime.KeepAlive(data)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_opp_cancel_transfer", &_hbcOppCancelTransfer, func(addr *deviceIDNative, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcOppCancelTransfer.funAddr(), uintptr(unsafe.Pointer(addr)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_opp_start_server", &_hbcOppStartServer, func(addr *bdAddr, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcOppStartServer.funAddr(), uintptr(unsafe.Pointer(addr)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hbc_opp_stop_server", &_hbcOppStopServer, func(addr *bdAddr, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbcOppStopServer.funAddr(), uintptr(unsafe.Pointer(addr)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(addr)
			runtime.KeepAlive(hberr)
			return ret
		}),
	}
}
