// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd

package lib

import "github.com/ebitengine/purego"

// OpenLibrary loads the library.
func OpenLibrary(name string) (uintptr, error) {
	return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

// CloseLibrary unloads the library.
func CloseLibrary(handle uintptr) error {
	return purego.Dlclose(handle)
}

// OpenSymbol loads a symbol from the library handle.
func OpenSymbol(lib uintptr, name string) (uintptr, error) {
	return purego.Dlsym(lib, name)
}
