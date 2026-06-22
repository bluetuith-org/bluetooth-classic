//go:build !linux && libhbluetooth

package lib

import ffi "github.com/bluetuith-org/libffi-go"

type nativeArray[T any] struct {
	List  ***T
	Count uint32
}

func newNativeArray[T any]() *nativeArray[T] {
	return &nativeArray[T]{}
}

func (n *nativeArray[T]) free(freeFun ffi.Fun) {
	freeFun.Call(nil, &n)
}
