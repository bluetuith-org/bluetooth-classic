//go:build !linux && libhbluetooth

package lib

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/appfeatures"
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	bcfg "github.com/bluetuith-org/bluetooth-classic/api/config"
	"github.com/bluetuith-org/bluetooth-classic/api/helpers/sessionstore"
	"github.com/ebitengine/purego"
)

type propAttributes uint32

type guid struct {
	i  uint32
	s1 uint16
	s2 uint16

	data [8]byte
}

var _libHandle = newLibHandle()

type libHandle struct {
	h      uintptr
	inited bool

	exitCh        chan int32
	waitForExitCh chan struct{}

	authorizer bluetooth.SessionAuthorizer
	store      *sessionstore.SessionStore

	mu sync.Mutex
}

func newLibHandle() *libHandle {
	return &libHandle{}
}

func (l *libHandle) initLibrary(store *sessionstore.SessionStore, authorizer bluetooth.SessionAuthorizer, cfg bcfg.Configuration) error {
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
			return errors.New("unsupported OS")
		}
	}

	handle, err := OpenLibrary(libName)
	if err != nil {
		return err
	}

	l.h = handle
	l.inited = true

	l.exitCh = make(chan int32, 1)
	l.waitForExitCh = make(chan struct{}, 1)

	l.authorizer = authorizer
	l.store = store

	for _, funInits := range [][]funHandle{
		getLibraryFunHandles(),
		getErrorFunHandles(),
		getAdapterFunHandles(),
		getDeviceFunHandles(),
		getOppFunHandles(),
	} {
		for _, initer := range funInits {
			var err error
			var fn uintptr

			fn, err = OpenSymbol(l.h, initer.funcName())
			if err != nil {
				return err
			}

			initer.setFunAddr(fn)
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

	nativeCbArg := eventCb

	ret := _hbSetEventCallbacks.Call(nativeCbArg, libErr.getHbErrorPtr())

	return libErr.getError(ret)
}

func (l *libHandle) removeEventHandlers() error {
	return nil
}

func (l *libHandle) closeLibrary(fn func()) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.inited {
		return nil
	}

	fn()

	if err := CloseLibrary(l.h); err != nil {
		return err
	}

	l.inited = false

	return l.removeEventHandlers()
}

// Initialize loads and initializes the library
func Initialize(store *sessionstore.SessionStore, authorizer bluetooth.SessionAuthorizer, cfg bcfg.Configuration) error {
	initChan := make(chan error, 2)

	setLaunched := func(err error) {
		initChan <- err
	}
	setExitWaited := func() {
		_libHandle.waitForExitCh <- struct{}{}
	}

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := _libHandle.initLibrary(store, authorizer, cfg); err != nil {
			initChan <- err
			return
		}

		cb := purego.NewCallback(func() uintptr {
			setLaunched(nil)

			return uintptr(<-_libHandle.exitCh)
		})

		libErr := newLibError()

		ret := _hbRunMain.Call(cb, libErr.getHbErrorPtr())

		setLaunched(libErr.getError(ret))
		setExitWaited()
	}()

	return <-initChan
}

// Release releases the library handle that was acquired via [Initialize].
func Release() {
	_libHandle.closeLibrary(func() {
		_libHandle.exitCh <- 0
		<-_libHandle.waitForExitCh
	})
}

// GetFeatures gets the supported features of the library.
func GetFeatures() appfeatures.Features {
	return appfeatures.Features(_hbGetFeatures.Call())
}

var (
	_hbRunMain           interopFunc[func(uintptr, **hbError) hbStatus]
	_hbGetFeatures       interopFunc[func() uint32]
	_hbSetEventCallbacks interopFunc[func(*eventCallbacks, **hbError) hbStatus]
	_hbSetAuthResponse   interopFunc[func(uint32, unsafe.Pointer, **hbError) hbStatus]
)

func getLibraryFunHandles() []funHandle {
	return []funHandle{
		newInteropFunc("hb_run_main", &_hbRunMain, func(mainFunc uintptr, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbRunMain.funAddr(), mainFunc, uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hb_get_features", &_hbGetFeatures, func() uint32 {
			_r0, _, _ := purego.SyscallN(_hbGetFeatures.funAddr())
			ret := uint32(_r0)
			return ret
		}),
		newInteropFunc("hb_set_events_cb", &_hbSetEventCallbacks, func(cb *eventCallbacks, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbSetEventCallbacks.funAddr(), uintptr(unsafe.Pointer(cb)), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(cb)
			runtime.KeepAlive(hberr)
			return ret
		}),
		newInteropFunc("hb_set_auth_response", &_hbSetAuthResponse, func(authID uint32, response unsafe.Pointer, hberr **hbError) hbStatus {
			_r0, _, _ := purego.SyscallN(_hbSetAuthResponse.funAddr(), uintptr(authID), uintptr(response), uintptr(unsafe.Pointer(hberr)))
			ret := hbStatus(_r0)
			runtime.KeepAlive(response)
			runtime.KeepAlive(hberr)
			return ret
		}),
	}
}
