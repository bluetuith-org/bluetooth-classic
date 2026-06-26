//go:build !linux && libhbluetooth

package lib

type funHandle interface {
	funAddr() uintptr
	setFunAddr(addr uintptr)
	funcName() string
}

type interopFunc[Fn any] struct {
	addr uintptr
	name string
	Call Fn

	store *interopFunc[Fn]
}

func newInteropFunc[Fn any](name string, store *interopFunc[Fn], fn Fn) *interopFunc[Fn] {
	return &interopFunc[Fn]{Call: fn, name: name, store: store}
}

func (i *interopFunc[Fn]) funAddr() uintptr {
	return i.addr
}

func (i *interopFunc[Fn]) setFunAddr(addr uintptr) {
	i.addr = addr
	*i.store = *i
}

func (i *interopFunc[Fn]) funcName() string {
	return i.name
}
