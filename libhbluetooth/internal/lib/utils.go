//go:build !linux && libhbluetooth

package lib

import (
	"strings"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/optional"
	"github.com/google/uuid"
)

func resolveDataPointer[T any](ptr unsafe.Pointer) *T {
	return *(**T)(ptr)
}

func bytePtrToString(p *byte) string {
	if p == nil {
		return ""
	}
	if *p == 0 {
		return ""
	}

	// Find NUL terminator.
	n := 0
	for ptr := unsafe.Pointer(p); *(*byte)(ptr) != 0; n++ {
		ptr = unsafe.Add(ptr, 1)
	}

	return string(unsafe.Slice(p, n))
}

func stringToBytePtr(p string) *byte {
	var sb strings.Builder

	sb.WriteString(p)
	sb.WriteString("\x00")

	return &[]byte(sb.String())[0]
}

func ptrToUUIDs(p *uuid.UUID, count uint32) uuid.UUIDs {
	if count == 0 {
		return nil
	}
	if p == nil {
		return nil
	}

	arr := unsafe.Slice((*uuid.UUID)(unsafe.Pointer(p)), count)

	return arr
}

func checkAndSetAttrs[T ~uint32](prop T, attrs uint32, fn func()) {
	if (attrs & uint32(prop)) == 0 {
		return
	}

	fn()
}

func optSetFunc[T optional.OptAllowed](set *optional.Optional[T], val T) func() {
	return func() {
		set.Set(val)
	}
}
