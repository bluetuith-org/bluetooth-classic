//go:build !linux && libhbluetooth

package lib

import (
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	ffi "github.com/bluetuith-org/libffi-go"
)

var (
	fnRetType = ffi.TypeSint32
	fnErrType = ffi.TypePointer
)

type propAttributes uint32

type bdAddr struct {
	Data bluetooth.MacAddress
}

func newBdAddr(address bluetooth.MacAddress) *bdAddr {
	addr := newBdAddrValue(address)

	return &addr
}

func newBdAddrValue(address bluetooth.MacAddress) bdAddr {
	return bdAddr{Data: address}
}
