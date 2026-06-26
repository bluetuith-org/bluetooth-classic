//go:build !linux && libhbluetooth

package lib

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
)

type hbStatus int32

const (
	statusOk             hbStatus = 0
	statusErr            hbStatus = -1
	statusErrNotReleased hbStatus = -2
)

type hbError struct {
	Category, Code uint32

	Description           *byte
	AdditionalInformation *byte
}

type libError struct {
	hberr *hbError
}

func newLibError() *libError {
	return &libError{}
}

func (l *libError) getHbErrorPtr() **hbError {
	h := &l.hberr

	return h
}

func (l *libError) getError(ret hbStatus) error {
	if ret == statusOk && l.hberr == nil {
		return nil
	}

	h := l.hberr
	if h != nil {
		var sb strings.Builder

		defer _hbErrorFree.Call(h)

		desc := bytePtrToString(l.hberr.Description)
		info := bytePtrToString(l.hberr.AdditionalInformation)

		if desc != "" {
			sb.WriteString(desc)
			sb.WriteString(" ")
		}
		if info != "" {
			fmt.Fprintf(&sb, "(%s)", info)
		}
		if desc == "" && info == "" {
			fmt.Fprintf(&sb, "Generic error: Category %d, Code %d", l.hberr.Category, l.hberr.Code)
		}

		return errors.New(sb.String())
	}

	return fmt.Errorf("generic error: Return code was %d", ret)
}

var _hbErrorFree interopFunc[func(*hbError)]

func getErrorFunHandles() []funHandle {
	return []funHandle{
		newInteropFunc("hb_error_free", &_hbErrorFree, func(hberr *hbError) {
			_, _, _ = purego.SyscallN(_hbErrorFree.funAddr(), uintptr(unsafe.Pointer(hberr)))
			runtime.KeepAlive(hberr)
		}),
	}
}
