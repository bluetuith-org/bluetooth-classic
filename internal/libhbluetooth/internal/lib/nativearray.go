//go:build !linux && libhbluetooth

package lib

type nativeArray[T any] struct {
	List  **T
	Count uint32
}

func newNativeArray[T any]() *nativeArray[T] {
	return &nativeArray[T]{}
}
