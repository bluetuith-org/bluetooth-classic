//go:build !linux && libhbluetooth

package lib

import (
	"errors"
	"fmt"
	"strings"

	ffi "github.com/bluetuith-org/libffi-go"
)

type hbError struct {
	Category, Code uint32

	Description           *byte
	AdditionalInformation *byte
}

type libError struct {
	ret   int32
	hberr *hbError
}

func newLibError() *libError {
	return &libError{}
}

func (l *libError) getReturnPtr() *int32 {
	return &l.ret
}

func (l *libError) getHbErrorPtr() ***hbError {
	h := &l.hberr

	return &h
}

func (l *libError) getError() error {
	if l.ret == 0 && l.hberr == nil {
		return nil
	}

	h := l.hberr
	if h != nil {
		var sb strings.Builder

		defer _hbErrorFree.Call(nil, &h)

		desc := bytePtrToString(l.hberr.Description)
		info := bytePtrToString(l.hberr.AdditionalInformation)

		sb.WriteString("error: ")
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

	return fmt.Errorf("generic error: Return code was %d", l.ret)
}

var _hbErrorFree ffi.Fun

func getErrorFunHandles() []funHandle {
	return []funHandle{
		{
			&_hbErrorFree, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hb_error_free", &ffi.TypeVoid, &ffi.TypePointer)
			},
		},
	}
}
