//go:build !linux && libhbluetooth

package lib

import (
	"context"
	"errors"
	"time"
	"unsafe"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/puzpuzpuz/xsync/v3"
)

type authEventType uint32

const (
	authEventNone authEventType = iota
	authEventCancelled
	authEventPairing
	authEventObjectPush
)

type authType uint32

const (
	authTypeNone authType = iota
	authTypeDisplayPinCode
	authTypeDisplayPassKey
	authTypeConfirmPasskey
	authTypeAuthorizePairing
	authTypeAuthorizeService
	authTypeAuthorizeTransfer
)

type authReplyMethod uint32

const (
	peplyMethodNone authReplyMethod = iota
	replyMethodConfirm
	replyMethodWithInput
)

var authMap = xsync.NewMapOf[uint32, *bluetooth.AuthTimeout]()

type authRequest[T any] struct {
	ID   uint32
	Data *T
}

type authResponse[T any] struct {
	Data *T
}

func newAuthResponse[E authResponse[T], T any]() *E {
	return &E{}
}

func (a *authResponse[T]) set(data T) *authResponse[T] {
	a.Data = &data

	return a
}

type cancelledAuthData struct{}

type pairingRequestData struct {
	Type        authType
	ReplyMethod authReplyMethod
	Timeout     uint32

	DeviceID deviceIDNative
	Passkey  uint32
	Pincode  [16]byte
}

type pairingResponseData struct {
	Passkey uint32
	Pincode [16]byte
}

type obexResponseData struct {
	Confirm bool
}

type (
	authRequestCancelled = authRequest[cancelledAuthData]
	authRequestPairing   = authRequest[pairingRequestData]
	authRequestOpp       = authRequest[oppTransferData]
)

type (
	authResponsePairing = authResponse[pairingResponseData]
	authResponseObex    = authResponse[obexResponseData]
)

func getAuthRequestData[E authRequest[T], T any](ptr unsafe.Pointer) (uint32, *T) {
	authReq := resolveDataPointer[E](ptr)
	auth := authRequest[T](*authReq)

	return auth.ID, auth.Data
}

func handleAuthEvent(etype authEventType, request unsafe.Pointer) {
	authorizer := _libHandle.authorizer

	switch etype {
	case authEventCancelled:
		authID, _ := getAuthRequestData[authRequestCancelled](request)
		if authTimeout, ok := authMap.LoadAndDelete(authID); ok {
			authTimeout.Cancel()
		}

	case authEventPairing:
		authID, dataPtr := getAuthRequestData[authRequestPairing](request)
		data := *dataPtr

		raiseAuthRequest(authID, func(ctx bluetooth.AuthTimeout) {
			var err error

			switch data.Type {
			case authTypeDisplayPinCode:
				err = authorizer.DisplayPinCode(ctx, string(data.Pincode[:]), data.DeviceID.ToDeviceAddress())

			case authTypeDisplayPassKey:
				err = authorizer.DisplayPasskey(ctx, data.Passkey, 0, data.DeviceID.ToDeviceAddress())

			case authTypeConfirmPasskey:
				err = authorizer.ConfirmPasskey(ctx, data.Passkey, data.DeviceID.ToDeviceAddress())

			case authTypeAuthorizePairing:
				err = authorizer.AuthorizePairing(ctx, data.DeviceID.ToDeviceAddress())

			case authTypeAuthorizeService:
				// ignored, unsupported
				return
			}

			var responseData pairingResponseData
			if err != nil && errors.Is(err, context.Canceled) {
				return
			}
			if err == nil {
				responseData = pairingResponseData{Passkey: data.Passkey, Pincode: data.Pincode}
			}

			response := newAuthResponse[authResponsePairing]().set(responseData)
			respondToAuthRequest(authID, response)
		})

	case authEventObjectPush:
		_ = authTypeAuthorizeTransfer

		authID, dataPtr := getAuthRequestData[authRequestOpp](request)
		data := dataPtr.toObjectPushData()

		raiseAuthRequest(authID, func(ctx bluetooth.AuthTimeout) {
			confirm := true

			if err := authorizer.AuthorizeTransfer(ctx, data); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}

				confirm = false
			}

			response := newAuthResponse[authResponseObex]().set(obexResponseData{confirm})
			respondToAuthRequest(authID, response)
		})
	}
}

func raiseAuthRequest(authID uint32, fn func(ctx bluetooth.AuthTimeout)) {
	go func() {
		timeoutCtx := bluetooth.NewAuthTimeout(10 * time.Second)

		authMap.Store(authID, &timeoutCtx)
		defer authMap.Delete(authID)

		fn(timeoutCtx)
		timeoutCtx.Cancel()
	}()
}

func respondToAuthRequest(authID uint32, response any) error {
	libErr := newLibError()

	_hbSetAuthResponse.Call(libErr.getReturnPtr(), &authID, response, libErr.getHbErrorPtr())

	return libErr.getError()
}
