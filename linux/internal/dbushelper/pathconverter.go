//go:build linux

package dbushelper

import (
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/godbus/dbus/v5"
	"github.com/puzpuzpuz/xsync/v3"
)

// DbusDevicePathType represents the type of DBus path in the Bluez DBus service.
// For example, adapter paths will have a path type of DbusPathAdapter and will
// be mapped to an adapter address (/org/bluez/hci0 => DBusPathAdapter).
// For other DBus path types like DbusPathObexSession and DbusPathObexTransfer,
// their paths will be mapped to device addresses.
type DbusDevicePathType int

// The different Bluez DBus path types.
const (
	DbusPathDevice DbusDevicePathType = iota
	DbusPathObexSession
	DbusPathObexTransfer
)

// dbusPath holds the Bluez DBus path and its type.
type dbusPath struct {
	pathType DbusDevicePathType
	path     dbus.ObjectPath
}

// dbusPathConverter holds a list of Bluez DBus paths and maps them
// to their respective Bluetooth addresses.
type dbusPathConverter struct {
	adapterPaths *xsync.MapOf[dbusPath, bluetooth.AdapterAddress]
	devicePaths  *xsync.MapOf[dbusPath, bluetooth.DeviceAddress]
}

var dbusPathAdapter = DbusDevicePathType(-1)

// PathConverter is used to obtain respective Bluetooth addresses that are mapped to
// Bluez DBus paths. This is mainly used to identify adapters and devices.
var PathConverter = dbusPathConverter{
	adapterPaths: xsync.NewMapOf[dbusPath, bluetooth.AdapterAddress](),
	devicePaths:  xsync.NewMapOf[dbusPath, bluetooth.DeviceAddress](),
}

// AdapterAddress returns a adapter's Bluetooth address that is mapped to the provided Bluez DBus path.
func (d *dbusPathConverter) AdapterAddress(path dbus.ObjectPath) (bluetooth.AdapterAddress, bool) {
	return d.adapterPaths.Load(dbusPath{pathType: dbusPathAdapter, path: path})
}

// AddAdapterDbusPath adds a mapping of a adapter's Bluez DBus path and a Bluetooth address to the path converter.
func (d *dbusPathConverter) AddAdapterDbusPath(path dbus.ObjectPath, address bluetooth.AdapterAddress) {
	d.adapterPaths.Store(dbusPath{pathType: dbusPathAdapter, path: path}, address)
}

// RemoveAdapterDbusPath removes a mapping of a adapter's Bluez DBus path and a Bluetooth address from the path converter.
func (d *dbusPathConverter) RemoveAdapterDbusPath(path dbus.ObjectPath) {
	d.adapterPaths.Delete(dbusPath{pathType: dbusPathAdapter, path: path})
}

// AdapterDbusPath returns a adapter's Bluez DBus path that is mapped to the provided Bluetooth address.
func (d *dbusPathConverter) AdapterDbusPath(address bluetooth.AdapterAddress) (dbus.ObjectPath, bool) {
	var apath dbus.ObjectPath

	d.adapterPaths.Range(func(p dbusPath, a bluetooth.AdapterAddress) bool {
		if a == address {
			apath = p.path
			return false
		}

		return true
	})

	return apath, apath != ""
}

// DeviceAddress returns a device's Bluetooth address that is mapped to the provided Bluez DBus path.
func (d *dbusPathConverter) DeviceAddress(pathType DbusDevicePathType, path dbus.ObjectPath) (bluetooth.DeviceAddress, bool) {
	return d.devicePaths.Load(dbusPath{pathType: pathType, path: path})
}

// AddDeviceDbusPath adds a mapping of a device's Bluez DBus path and a Bluetooth address to the path converter.
func (d *dbusPathConverter) AddDeviceDbusPath(pathType DbusDevicePathType, path dbus.ObjectPath, address bluetooth.DeviceAddress) {
	d.devicePaths.Store(dbusPath{pathType: pathType, path: path}, address)
}

// RemoveDeviceDbusPath removes a mapping of a device's Bluez DBus path and a Bluetooth address from the path converter.
func (d *dbusPathConverter) RemoveDeviceDbusPath(pathType DbusDevicePathType, path dbus.ObjectPath) {
	d.devicePaths.Delete(dbusPath{pathType: pathType, path: path})
}

// DeviceDbusPath returns a device's Bluez DBus path that is mapped to the provided Bluetooth address.
func (d *dbusPathConverter) DeviceDbusPath(pathType DbusDevicePathType, address bluetooth.DeviceAddress) (dbus.ObjectPath, bool) {
	var dpath dbus.ObjectPath

	d.devicePaths.Range(func(p dbusPath, a bluetooth.DeviceAddress) bool {
		if a == address && p.pathType == pathType {
			dpath = p.path

			return false
		}

		return true
	})

	return dpath, dpath != ""
}
