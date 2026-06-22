//go:build !linux && libhbluetooth

package lib

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/appfeatures"
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	bcfg "github.com/bluetuith-org/bluetooth-classic/api/config"
	ffi "github.com/bluetuith-org/libffi-go"
)

type funHandle struct {
	fun        *ffi.Fun
	handleFunc func(handle ffi.Lib, fun *ffi.Fun, err *error)
}

var _libHandle = newLibHandle()

type libHandle struct {
	h      ffi.Lib
	inited bool

	eventCb map[*eventCallbacks]struct{}

	exitCh        chan int32
	waitForExitCh chan struct{}

	authorizer bluetooth.SessionAuthorizer

	mu sync.Mutex
}

func newLibHandle() *libHandle {
	return &libHandle{
		eventCb: make(map[*eventCallbacks]struct{}),
	}
}

func (l *libHandle) initLibrary(authorizer bluetooth.SessionAuthorizer, cfg bcfg.Configuration) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.inited {
		return nil
	}

	libName := cfg.LibraryPath
	if libName == "" {
		libName = "libhbluetooth"
		switch runtime.GOOS {
		case "windows":
			libName += ".dll"

		case "darwin":
			libName += ".dylib"

		default:
			panic("Unsupported OS")
		}
	}

	handle, err := ffi.LoadWithSymbols(libName)
	if err != nil {
		return err
	}

	l.h = handle
	l.inited = true

	l.exitCh = make(chan int32, 1)
	l.waitForExitCh = make(chan struct{}, 1)

	l.authorizer = authorizer

	for _, funInits := range [][]funHandle{
		getLibraryFunHandles(),
		getErrorFunHandles(),
		getAdapterFunHandles(),
		getDeviceFunHandles(),
		getOppFunHandles(),
	} {
		for _, initer := range funInits {
			var err error

			initer.handleFunc(handle, initer.fun, &err)
			if err != nil {
				return err
			}
		}
	}

	return l.addEventHandlers()
}

func (l *libHandle) addEventHandlers() error {
	libErr := newLibError()

	eventCb, err := newEventCallbacks()
	if err != nil {
		return err
	}

	nativeCbArg := eventCb.toNativeCallbacks()

	_hbSetEventCallbacks.Call(libErr.getReturnPtr(), &nativeCbArg, libErr.getHbErrorPtr())
	if err := libErr.getError(); err != nil {
		return err
	}

	l.eventCb[eventCb] = struct{}{}

	return nil
}

func (l *libHandle) removeEventHandlers() error {
	// TODO: Free event callbacks

	l.eventCb = nil

	return nil
}

func (l *libHandle) closeLibrary(fn func()) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.inited {
		return nil
	}

	fn()

	if err := l.h.Close(); err != nil {
		return err
	}

	l.inited = false

	return l.removeEventHandlers()
}

// Initialize loads and initializes the library
//
//revive:disable
func Initialize(authorizer bluetooth.SessionAuthorizer, cfg bcfg.Configuration) error {
	if err := _libHandle.initLibrary(authorizer, cfg); err != nil {
		return err
	}

	initChan := make(chan error, 2)

	setLaunched := func(err error) {
		initChan <- err
	}
	setExitWaited := func() {
		_libHandle.waitForExitCh <- struct{}{}
	}

	cb, err := newCallback(func(_ *ffi.Cif, ret unsafe.Pointer, _ *unsafe.Pointer, _ unsafe.Pointer) uintptr {
		setLaunched(nil)
		*(*int32)(ret) = <-_libHandle.exitCh

		return 0
	}, ffi.DefaultAbi, 0, &ffi.TypeSint32)
	if err != nil {
		return err
	}

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		defer cb.free()

		libErr := newLibError()

		_hbRunMain.Call(libErr.getReturnPtr(), cb.getCallBackPtr(), libErr.getHbErrorPtr())

		setLaunched(libErr.getError())
		setExitWaited()
	}()

	return <-initChan
}

//revive:enable

// Release releases the library handle that was acquired via [Initialize].
func Release() {
	_libHandle.closeLibrary(func() {
		_libHandle.exitCh <- 0
		<-_libHandle.waitForExitCh
	})
}

// GetFeatures gets the supported features of the library.
func GetFeatures() appfeatures.Features {
	var f appfeatures.Features

	_hbGetFeatures.Call(&f)

	return f
}

var (
	_hbRunMain           ffi.Fun
	_hbSetEventCallbacks ffi.Fun
	_hbSetAuthResponse   ffi.Fun
	_hbGetFeatures       ffi.Fun
)

func getLibraryFunHandles() []funHandle {
	return []funHandle{
		{
			&_hbGetFeatures, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hb_get_features", &ffi.TypeUint32)
			},
		},
		{
			&_hbSetEventCallbacks, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hb_set_events_cb", &fnRetType, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbRunMain, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hb_run_main", &ffi.TypeSint32, &ffi.TypePointer, &fnErrType)
			},
		},
		{
			&_hbSetAuthResponse, func(handle ffi.Lib, fun *ffi.Fun, err *error) {
				*fun, *err = handle.Prep("hb_set_auth_response", &fnRetType, &ffi.TypeUint32, &ffi.TypePointer, &fnErrType)
			},
		},
	}
}
