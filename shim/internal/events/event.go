//go:build !linux

package events

import (
	"github.com/bluetuith-org/bluetooth-classic/api/bluetooth"
	"github.com/bluetuith-org/bluetooth-classic/shim/internal/serde"
	"github.com/ugorji/go/codec"
)

// ServerEvent describes a raw event that was sent from the server.
type ServerEvent struct {
	EventId     bluetooth.EventID     `json:"event_id,omitempty"`
	EventAction bluetooth.EventAction `json:"event_action"`
	Event       codec.Raw             `json:"event"`
}

// UnmarshalBluetoothEvent unmarshals a 'ServerEvent' to a bluetooth event.
func UnmarshalBluetoothEvent[T bluetooth.Events](ev ServerEvent) (bluetooth.Event[T], error) {
	var event bluetooth.Event[T]

	unmarshalled := make(map[string]T, 1)

	if err := serde.UnmarshalJson(ev.Event, &unmarshalled); err != nil {
		return event, err
	}

	event.ID = ev.EventId
	event.Action = ev.EventAction
	for _, m := range unmarshalled {
		event.Data = m
	}

	return event, nil
}
