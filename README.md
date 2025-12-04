[![Go Reference](https://pkg.go.dev/badge/github.com/bluetuith-org/bluetooth-classic.svg)](https://pkg.go.dev/github.com/bluetuith-org/bluetooth-classic)

# bluetooth-classic

This is a library which provides multi-platform support to handle Bluetooth Classic functionality.
Since this is an alpha release, expect the API to change at any time.

Currently, support is present for the following platforms:
- Linux
- Windows

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

- Windows
  - [haraltd](https://github.com/bluetuith-org/haraltd)
  - Start the server before using this library (i.e using the `server start` command)

## Feature matrix
|Features (APIs) / OS|Pairing            |Connection (Automatic/Profile-based)|Send/Receive Files (OBEX Object PUSH)|Network Tethering (PANU/DUN)|Media Control (AVRCP)|
|--------------------|-------------------|------------------------------------|-------------------------------------|----------------------------|---------------------|
|Linux               |:heavy_check_mark: |:heavy_check_mark: (Yes/Yes)        |:heavy_check_mark:                   |:heavy_check_mark:          |:heavy_check_mark:   |
|Windows             |:heavy_check_mark: |:heavy_check_mark: (Yes/Yes)        |:heavy_check_mark:                   |Not yet implemented         |Not yet implemented  |
|MacOS               |:heavy_check_mark: |:heavy_check_mark: (Yes/No)         |:heavy_check_mark:                   |Not yet implemented         |Not yet implemented  |
|FreeBSD             |Not yet implemented|Not yet implemented                 |Not yet implemented                  |Not yet implemented         |Not yet implemented  |

## Documentation
The documentation is not very extensive right now, more documentation will be added later.
See the package reference for more information.

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

    // Create a new session.
	session := session.NewSession()

    // Attempt to start the session, and if it returns with no errors,
    // it will provide the supported features and platform information of the session.
    // The autHandler struct handles all authentication requests, like for pairing or file transfer.
	featureSet, pinfo, err := session.Start(&authHandler{}, cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer session.Stop()

    // Listen for any device events when devices are added to or removed from the system or its properties are updated.
	go func() {
		id := bluetooth.DeviceEvent().Subscribe()
		if !id.Subscribable {
			return
		}

		for device := range id.C {
			fmt.Println(device.Data)
		}
	}()

    // Listen for any adapter events when adapters are added to or removed from the system or its properties are updated.
    go func() {
		id := bluetooth.AdapterEvent().Subscribe()
		if !id.Subscribable {
			return
		}

		for adapter := range id.C {
			fmt.Println(adapter.Data)
		}
	}()

    // Print the platform information.
	fmt.Println(pinfo)

    // Print the supported features of the session.
	fmt.Println(featureSet.Supported.String())
    
    // Get a list of adapters.
    adapters := session.Adapters()
	for _, adapter := range adapters {
		fmt.Println(adapter.Name)
	}
	if len(adapters) == 0 {
		panic("no adapters found")
	}

    // Select an adapter from the list.
	selectedAdapter := session.Adapter(adapters[0].Address)

    // Start discovering devices on the selected adapter.
    // All discovered devices will be sent as device events.
	err = selectedAdapter.StartDiscovery()
	if err != nil {
		panic(err)
	}

	time.Sleep(10000)

    // Stop discovering devices on the selected adapter.
	err = selectedAdapter.StopDiscovery()
	if err != nil {
		panic(err)
	}
}

// authHandler provides a custom SessionAuthorizer based implementation to handle authentication requests,
// like for pairing or incoming file transfers. For each of the handler functions, return an error if
// the authentication request should be denied.
type authHandler struct {
}

func (a *authHandler) AuthorizeTransfer(timeout bluetooth.AuthTimeout, props bluetooth.ObjectPushData) error {
    fmt.Println(props)
	return nil
}

func (a *authHandler) DisplayPinCode(timeout bluetooth.AuthTimeout, address bluetooth.MacAddress, pincode string) error {
    return errors.New("do not authenticate")
}

func (a *authHandler) DisplayPasskey(timeout bluetooth.AuthTimeout, address bluetooth.MacAddress, passkey uint32, entered uint16) error {
    fmt.Println(passkey)
	return nil
}

func (a *authHandler) ConfirmPasskey(timeout bluetooth.AuthTimeout, address bluetooth.MacAddress, passkey uint32) error {
	fmt.Println(passkey)

	return nil
}

func (a *authHandler) AuthorizePairing(timeout bluetooth.AuthTimeout, address bluetooth.MacAddress) error {
	return nil
}

func (a *authHandler) AuthorizeService(timeout bluetooth.AuthTimeout, address bluetooth.MacAddress, uuid uuid.UUID) error {
	return nil
}
```



  

