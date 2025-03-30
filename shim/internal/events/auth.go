//go:build !linux

package events

import (
	"errors"
	"strconv"
	"time"

	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/bluetuith-org/bluetooth-classic/shim/internal/serde"
	"github.com/google/uuid"
)

// AuthEventID describes the type of authentication to be performed.
type AuthEventID string

// The various authentication event ID types.
const (
	DisplayPinCode    AuthEventID = "display-pincode"
	DisplayPasskey    AuthEventID = "display-passkey"
	ConfirmPasskey    AuthEventID = "confirm-passkey"
	AuthorizePairing  AuthEventID = "authorize-pairing"
	AuthorizeService  AuthEventID = "authorize-service"
	AuthorizeTransfer AuthEventID = "authorize-transfer"
)

// AuthReplyMethod describes a method to reply to an authentication request.
type AuthReplyMethod string

// The various authentication reply types.
const (
	ReplyNone      AuthReplyMethod = "reply-none"
	ReplyYesNo     AuthReplyMethod = "reply-yes-no"
	ReplyWithInput AuthReplyMethod = "reply-with-input"
)

// AuthReply wraps a reply method and its associated reply.
type AuthReply struct {
	ReplyMethod AuthReplyMethod
	Reply       string
}

// AuthEventData describes an authentication event.
type AuthEventData struct {
	AuthID      int             `json:"auth_id,omitempty"`
	EventID     AuthEventID     `json:"auth_event,omitempty"`
	ReplyMethod AuthReplyMethod `json:"auth_reply_method,omitempty"`

	TimeoutMs int                  `json:"timeout_ms,omitempty"`
	Address   bluetooth.MacAddress `json:"address,omitempty"`

	Pincode string `json:"pincode,omitempty"`

	Passkey uint32 `json:"passkey,omitempty"`
	Entered uint16 `json:"entered,omitempty"`

	UUID uuid.UUID `json:"uuid,omitempty"`

	FileTransfer bluetooth.FileTransferData `json:"file_transfer,omitempty"`
}

// CallAuthorizer maps the authentication event to the registered 'SessionAuthorizer' handlers.
func (a *AuthEventData) CallAuthorizer(authorizer bluetooth.SessionAuthorizer, cb func(authEvent AuthEventData, reply AuthReply, err error)) error {
	if authorizer == nil {
		return errors.New("authorizer cannot be nil")
	}

	var authfn func() (AuthReply, error)

	switch a.EventID {
	case DisplayPinCode:
		authfn = func() (AuthReply, error) {
			return AuthReply{ReplyWithInput, a.Pincode},
				authorizer.DisplayPinCode(bluetooth.NewAuthTimeout(time.Duration(a.TimeoutMs)), a.Address, a.Pincode)
		}

	case DisplayPasskey:
		authfn = func() (AuthReply, error) {
			return AuthReply{ReplyWithInput, strconv.FormatUint(uint64(a.Passkey), 10)},
				authorizer.DisplayPasskey(bluetooth.NewAuthTimeout(time.Duration(a.TimeoutMs)), a.Address, a.Passkey, a.Entered)
		}

	case ConfirmPasskey:
		authfn = func() (AuthReply, error) {
			return AuthReply{ReplyYesNo, "yes"},
				authorizer.ConfirmPasskey(bluetooth.NewAuthTimeout(time.Duration(a.TimeoutMs)), a.Address, a.Passkey)
		}

	case AuthorizePairing:
		authfn = func() (AuthReply, error) {
			return AuthReply{ReplyYesNo, "yes"},
				authorizer.AuthorizePairing(bluetooth.NewAuthTimeout(time.Duration(a.TimeoutMs)), a.Address)
		}

	case AuthorizeService:
		authfn = func() (AuthReply, error) {
			return AuthReply{ReplyYesNo, "yes"},
				authorizer.AuthorizeService(bluetooth.NewAuthTimeout(time.Duration(a.TimeoutMs)), a.Address, a.UUID)
		}

	case AuthorizeTransfer:
		authfn = func() (AuthReply, error) {
			return AuthReply{ReplyYesNo, "yes"},
				authorizer.AuthorizeTransfer(bluetooth.NewAuthTimeout(time.Duration(a.TimeoutMs)), a.FileTransfer)
		}
	}

	if authfn == nil {
		return errorkinds.ErrMethodCall
	}

	reply, err := authfn()
	cb(*a, reply, err)

	return nil
}

// UnmarshalAuthEvent unmarshals a 'ServerEvent' to an authentication event.
func UnmarshalAuthEvent(ev ServerEvent) (AuthEventData, error) {
	var event AuthEventData

	unmarshalled := make(map[string]AuthEventData, 1)

	if err := serde.UnmarshalJson(ev.Event, &unmarshalled); err != nil {
		return event, err
	}

	return event, nil
}
