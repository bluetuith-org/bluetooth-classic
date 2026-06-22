package platforminfo

import "runtime"

// PlatformInfo describes platform-specific information.
type PlatformInfo struct {
	OS             string `json:"os_info,omitempty"`
	Stack          string `json:"stack,omitempty"`
	Implementation string
}

// NewPlatformInfo returns a new PlatformInfo.
func NewPlatformInfo(stack, impl string) PlatformInfo {
	return PlatformInfo{
		OS:             runtime.GOOS + " (" + runtime.GOARCH + ")",
		Stack:          stack,
		Implementation: impl,
	}
}
