//go:build linux

package dbushelper

import (
	"strings"

	"github.com/godbus/dbus/v5"
)

// ListActivatableBusNames returns a list of bus names from the provided DBus connection.
func ListActivatableBusNames(conn *dbus.Conn) ([]string, error) {
	var names []string

	if err := conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").
		Call("org.freedesktop.DBus.ListActivatableNames", 0).
		Store(&names); err != nil {
		return nil, err
	}

	return names, nil
}

// GetAdapterPathFromSignal extracts the adapter path from the signal's path.
func GetAdapterPathFromSignal(signalPath dbus.ObjectPath) (dbus.ObjectPath, bool) {
	return extractPathFromSignal(signalPath, "hci")
}

// GetDevicePathFromSignal extracts the device path from the signal's path.
func GetDevicePathFromSignal(signalPath dbus.ObjectPath) (dbus.ObjectPath, bool) {
	return extractPathFromSignal(signalPath, "dev_")
}

// extractPathFromSignal extracts a part of a path from a complete signal path.
func extractPathFromSignal(signalPath dbus.ObjectPath, substr string) (dbus.ObjectPath, bool) {
	if signalPath == "" {
		return "", false
	}

	var sb strings.Builder

	found := false

	sp := strings.SplitAfterSeq(string(signalPath), "/")
	for str := range sp {
		if strings.Contains(str, substr) {
			found = true
			sb.WriteString(str[0 : len(str)-1])

			break
		}

		sb.WriteString(str)
	}

	return dbus.ObjectPath(sb.String()), found
}
