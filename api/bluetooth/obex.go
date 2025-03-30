package bluetooth

import (
	"context"
)

// Obex describes a function call interface to invoke Obex related functions
// on specified devices.
type Obex interface {
	// FileTransfer returns a function call interface to invoke device file transfer
	// related functions.
	FileTransfer() ObexFileTransfer
}

// ObexFileTransfer describes a function call interface to manage file-transfer
// related functions on specified devices.
type ObexFileTransfer interface {
	// CreateSession creates a new Obex session with a device.
	// The context (ctx) can be provided in case this function call
	// needs to be cancelled, since this function call can take some time
	// to complete.
	CreateSession(ctx context.Context) error

	// RemoveSession removes a created Obex session.
	RemoveSession() error

	// SendFile sends a file to the device. The 'filepath' must be a full path to the file.
	SendFile(filepath string) (FileTransferData, error)

	// CancelTransfer cancels the transfer.
	CancelTransfer() error

	// SuspendTransfer suspends the transfer.
	SuspendTransfer() error

	// ResumeTransfer resumes the transfer.
	ResumeTransfer() error
}

// FileTransferStatus describes the status of the file transfer.
type FileTransferStatus string

// The different transfer status types.
const (
	TransferQueued    FileTransferStatus = "queued"
	TransferActive    FileTransferStatus = "active"
	TransferSuspended FileTransferStatus = "suspended"
	TransferComplete  FileTransferStatus = "complete"
	TransferError     FileTransferStatus = "error"
)

// FileTransferData holds the static file transfer data for a device.
type FileTransferData struct {
	// Name is the name of the object being transferred.
	Name string `json:"name,omitempty" codec:"Name,omitempty" doc:"The name of the object being transferred."`

	// Type is the type of the file (mime-type).
	Type string `json:"type,omitempty" codec:"Type,omitempty" doc:"The type of the file (mime-type)."`

	// Status indicates the file transfer status.
	Status FileTransferStatus `json:"status,omitempty" codec:"Status,omitempty" enum:"queued,active,suspended,complete,error" doc:"Indicates the file transfer status."`

	// Filename is the complete name of the file.
	Filename string `json:"filename,omitempty" codec:"Filename,omitempty" doc:"The complete name of the file."`

	FileTransferEventData
}

// FileTransferEventData holds the dynamic (variable) file transfer data for a device.
// This is primarily used to send file transfer event related data.
type FileTransferEventData struct {
	// Address holds the Bluetooth MAC address of the device.
	Address MacAddress `json:"address,omitempty" codec:"Address,omitempty" doc:"The Bluetooth MAC address of the device."`

	// Size holds the total size of the file in bytes.
	Size uint64 `json:"size,omitempty" codec:"Size,omitempty" doc:"The total size of the file in bytes."`

	// Transferred holds the current number of bytes that was sent to the receiver.
	Transferred uint64 `json:"transferred,omitempty" codec:"Transferred,omitempty" doc:"The current number of bytes that was sent to the receiver."`
}

// AuthorizeReceiveFile describes an authentication interface, which is used
// to authorize a file transfer being received, before starting the transfer.
type AuthorizeReceiveFile interface {
	AuthorizeTransfer(timeout AuthTimeout, props FileTransferData) error
}
