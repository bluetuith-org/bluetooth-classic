[![Go Reference](https://pkg.go.dev/badge/github.com/bluetuith-org/bluetooth-classic.svg)](https://pkg.go.dev/github.com/bluetuith-org/bluetooth-classic)

# bluetooth-classic

This is a library which provides multi-platform support to handle Bluetooth Classic functionality.
Since this is an alpha release, expect the API to change at any time.

Currently, support is present for the following platforms:

- Linux
- Windows
- MacOS

## Funding

This project is funded through [NGI Zero Core](https://nlnet.nl/core), a fund established by [NLnet](https://nlnet.nl) with financial support from the European Commission's [Next Generation Internet](https://ngi.eu) program. Learn more at the [NLnet project page](https://nlnet.nl/project/bluetuith).

[<img src="https://nlnet.nl/logo/banner.png" alt="NLnet foundation logo" width="20%" />](https://nlnet.nl)
[<img src="https://nlnet.nl/image/logos/NGI0_tag.svg" alt="NGI Zero Logo" width="20%" />](https://nlnet.nl/core)

## Dependencies

- Linux
  - Bluez
  - DBus
  - NetworkManager (optional, required for PANU)
  - ModemManager (optional, required for DUN)
  - PulseAudio (optional, required to manage device audio profiles)
- Windows / MacOS
  - [haraltd](https://github.com/bluetuith-org/haraltd) or [libhbluetooth](https://github.com/bluetuith-org/libhbluetooth)
  - See the documentation for usage instructions.

## Feature matrix

|Features (APIs) / OS|Pairing            |Connection (Automatic/Profile-based)     |Send/Receive Files (OBEX Object PUSH)|Network Tethering (PANU/DUN)|Media Control (AVRCP)|
|--------------------|-------------------|------------------------------------     |-------------------------------------|----------------------------|---------------------|
|Linux               |:heavy_check_mark: |:heavy_check_mark: (Yes/Yes)             |:heavy_check_mark:                   |:heavy_check_mark:          |:heavy_check_mark:   |
|Windows             |:heavy_check_mark: |:heavy_check_mark: (Yes/No)              |:heavy_check_mark:                   |Not yet implemented         |Not yet implemented  |
|MacOS               |:heavy_check_mark: |:heavy_check_mark: (Yes/No)              |:heavy_check_mark:                   |Not yet implemented         |Not yet implemented  |

## Documentation

The API is documented within the package reference linked at the top of the README.

The following instructions describe how to build and use the library.

### Linux

Apart from the requirements listed above, no other setup is necessary. Simply:

- Import the project:
  `go get github.com/bluetuith-org/bluetooth-classic`

- Ensure that the 'BlueZ' service is running.<br />
  Optionally, enable the 'obexd' service, to enable Object Push transfers.

- Compile and run the program. See the sample usage below for an example.

### Windows/MacOS

Either one of [haraltd](https://github.com/bluetuith-org/haraltd) or [libhbluetooth](https://github.com/bluetuith-org/libhbluetooth) dependencies need to be installed.
It is not required to have both of them at the same time.

- Import the project:

  ```sh
  go get github.com/bluetuith-org/bluetooth-classic
  ```

Then, follow the instructions below, depending on what dependency is installed.

#### haraltd

- To build for `haraltd`, a build tag must be used.
  Compile the project as follows:

  ```sh
  go build -tags="haraltd" -o <executable-name> .
  ```

- Ensure that the daemon is started (i.e. call `haraltd server start`).

- Then run the program. See the sample usage below for an example.

#### libhbluetooth

- To build for `libhbluetooth`, a build tag must be used.
  Compile the project as follows:

  ```sh
  go build -tags="libhbluetooth" -o <executable-name> .
  ```

- Ensure that the library is located in a well-known path. Ideally, it can be placed right next to
  the compiled executable. Or, to specify a custom path, use the configuration options described in the sample.

- Then run the program. See the sample usage below for an example.

## Sample Usage

```go
package main

import (
 "fmt"

 "github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
 "github.com/bluetuith-org/bluetooth-classic/api/config"
 "github.com/bluetuith-org/bluetooth-classic/session"
 "github.com/google/uuid"
)

func main() {
 // Initialize a session configuration.
 cfg := config.New()

 // You can specify a custom socket path to connect to,
 // provided that the 'haraltd' daemon is running and is configured
 // to run on a different socket path. If the daemon is started as-is,
 // this option is not required.
 // The library must be built with the 'haraltd' build tag for this option to
 // be valid.
 cfg.SocketPath = "./socketdir/custom.sock"

 // You can specify a custom path to load the 'libhbluetooth' dynamic library,
 // if the 'libhbluetooth' library is present on your system.
 // The library must be built with the 'libhbluetooth' build tag for this option to
 // be valid.
 cfg.LibraryPath = "./library/libhbluetooth.dll"

 // Create a new session.
 // If the library is built with the:
 // - the 'haraltd' build tag, it tries to create a new daemon session, OR
 // - the 'libhbluetooth' build tag, it tries to create a new interface with the 'libhbluetooth' dynamic library.
 classicSession := session.NewSession()

 // Attempt to start the session, and if it returns with no errors,
 // it will provide the supported features and platform information of the session.
 // The autHandler struct handles all authentication requests, like for pairing or file transfer.
 featureSet, pinfo, err := classicSession.Start(&authHandler{}, cfg)
 if err != nil {
  fmt.Println(err)
  return
 }
 defer classicSession.Stop()

 // Listen for any device events when devices are added to or removed from the system or its properties are updated.
 go func() {
  id, ok := bluetooth.DeviceEvents().Subscribe()
  if !ok {
   return
  }

  for device := range id.UpdatedEvents {
   fmt.Println(device)
  }
 }()

 // Listen for any adapter events when adapters are added to or removed from the system or its properties are updated.
 go func() {
  id, ok := bluetooth.AdapterEvents().Subscribe()
  if !ok {
   return
  }

  for adapter := range id.UpdatedEvents {
   fmt.Println(adapter)
  }
 }()

 // Print the platform information.
 fmt.Println(pinfo)

 // Print the supported features of the session.
 fmt.Println(featureSet.Supported.String())

 // Get a list of adapters.
 adapters, err := classicSession.Adapters()
 if err != nil {
  panic(err)
 }
 for _, adapter := range adapters {
  fmt.Println(adapter.Name)
 }

 selectedAdapter := classicSession.Adapter(adapters[0].AdapterAddress)

 // Start discovering devices on the selected adapter.
 // All discovered devices will be sent as device events.
 err = selectedAdapter.StartDiscovery()
 if err != nil {
  panic(err)
 }

 id, ok := bluetooth.DeviceEvents().Subscribe()
 if !ok {
  panic("Could not subscribe to device events")
 }

 // Get the first device that was discovered.
 var foundDevice bluetooth.DeviceData
 for device := range id.AddedEvents {
  if paired, ok := device.Paired.Get(); ok && !paired {
   foundDevice = device
   break
  }
 }

 // This is where the configured SessionAuthorizer is called (here it is 'authHandler').
 // For pairing, any one of the *Pincode or *Passkey methods can be called, depending on the
 // type of pairing.
 //
 // Note: The authHandler's AuthorizeTransfer' method is only called
 // when an OBEX Object Push transfer has to be authorized, so it is not called here.
 if err := classicSession.Device(foundDevice.DeviceAddress).Pair(); err != nil {
  fmt.Println(err)
 }

 // Stop discovering devices on the selected adapter.
 err = selectedAdapter.StopDiscovery()
 if err != nil {
  panic(err)
 }
}

// authHandler provides a custom SessionAuthorizer based implementation to handle authentication requests,
// like for pairing or incoming file transfers. For each of the handler functions, return an error if
// the authentication request should be denied.
type authHandler struct{}

func (a *authHandler) AuthorizeTransfer(timeout bluetooth.AuthTimeout, props bluetooth.ObjectPushData) error {
 return nil
}

func (a *authHandler) DisplayPinCode(timeout bluetooth.AuthTimeout, pincode string, address bluetooth.DeviceAddress) error {
 return nil
}

func (a *authHandler) DisplayPasskey(timeout bluetooth.AuthTimeout, passkey uint32, entered uint16, address bluetooth.DeviceAddress) error {
 return nil
}

func (a *authHandler) ConfirmPasskey(timeout bluetooth.AuthTimeout, passkey uint32, address bluetooth.DeviceAddress) error {
 return nil
}

func (a *authHandler) AuthorizePairing(timeout bluetooth.AuthTimeout, address bluetooth.DeviceAddress) error {
 return nil
}

func (a *authHandler) AuthorizeService(timeout bluetooth.AuthTimeout, serviceUUID uuid.UUID, address bluetooth.DeviceAddress) error {
 return nil
}
```
  
