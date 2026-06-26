// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

//go:build windows

package lib

import (
	"syscall"
)

// OpenLibrary loads the library.
func OpenLibrary(name string) (uintptr, error) {
	handle, err := syscall.LoadLibrary(name)
	return uintptr(handle), err
}

// CloseLibrary unloads the library.
func CloseLibrary(handle uintptr) error {
	return syscall.FreeLibrary(syscall.Handle(handle))
}

// OpenSymbol loads a symbol from the library handle.
func OpenSymbol(lib uintptr, name string) (uintptr, error) {
	return syscall.GetProcAddress(syscall.Handle(lib), name)
}
