//go:build !linux && libhbluetooth

package lib

import (
	"encoding/binary"
	"strings"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/optional"
	"github.com/google/uuid"
)

func resolveDataPointer[T any](ptr unsafe.Pointer) *T {
	return (*T)(ptr)
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

func ptrToUUIDs(p *guid, count uint32) uuid.UUIDs {
	if count == 0 {
		return nil
	}
	if p == nil {
		return nil
	}

	guidArray := unsafe.Slice((*guid)(unsafe.Pointer(p)), count)

	uuids := make([]uuid.UUID, 0, count)
	order := binary.BigEndian

	for _, val := range guidArray {
		var uuidValue uuid.UUID

		order.PutUint32(uuidValue[0:4], val.i)
		order.PutUint16(uuidValue[4:6], val.s1)
		order.PutUint16(uuidValue[6:8], val.s2)
		copy(uuidValue[8:], val.data[:])

		uuids = append(uuids, uuidValue)
	}

	return uuids
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
