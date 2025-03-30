package bluetooth

import (
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/bluetuith-org/bluetooth-classic/api/eventbus"
)

// Events defines a set of possible event data types.
type Events interface {
	errorkinds.GenericError | AdapterEventData | DeviceEventData | MediaEventData | FileTransferEventData
}

// Event represents a general event.
type Event[T Events] struct {
	// ID holds the event ID.
	ID EventID `json:"event_id,omitempty" doc:"The event ID."`

	// Action holds the corresponding action associated
	// with this event.
	Action EventAction `json:"event_action,omitempty" enum:"updated,added,removed" doc:"The corresponding action associated with this event"`

	// Data holds the actual event data.
	Data T `json:"event_data,omitempty" doc:"The actual event data."`
}

// Subscriber describes a subscriber token.
type Subscriber[T Events] struct {
	C            <-chan Event[T]
	Subscribable bool
	Unsubscribe  eventbus.UnsubFunc
}

// EventID represents a unique event ID.
type EventID byte

// The different types of event IDs.
const (
	EventNone EventID = iota // The zero value for this type.
	EventError
	EventAdapter
	EventDevice
	EventFileTransfer
	EventMediaPlayer
	EventAuthentication
)

// EventAction describes an action that is associated with an event.
type EventAction string

// The different types of event actions.
const (
	EventActionNone    EventAction = "none"
	EventActionUpdated EventAction = "updated"
	EventActionAdded   EventAction = "added"
	EventActionRemoved EventAction = "removed"
)

// eventNames holds names of different events.
var (
	eventNames = map[EventID]string{
		EventNone:         "",
		EventError:        "error_event",
		EventAdapter:      "adapter_event",
		EventDevice:       "device_event",
		EventFileTransfer: "file_transfer_event",
		EventMediaPlayer:  "media_player_event",
	}
)

// String returns the name of the event ID.
func (e EventID) String() string {
	return eventNames[e]
}

// String returns the name of the event ID.
func (e EventAction) String() string {
	return string(e)
}

// Value returns the event ID.
func (e EventID) Value() uint {
	return uint(e)
}

// PublishData publishes the event to the event stream with the provided data.
func (e Event[T]) PublishData(data T) {
	e.Data = data
	eventbus.Publish(e.ID, e)
}

// Publish publishes the event to the event stream as-is.
func (e Event[T]) Publish() {
	eventbus.Publish(e.ID, e)
}

// Subscribe listens to the event stream and subscribes to the event.
// To unsubscribe from the event, use (Subscriber).Unsubscribe().
// It will never return a nil channel.
// To check if the returned channel will get events, use (Subscriber).Subscribable.
func (e Event[T]) Subscribe() Subscriber[T] {
	eventChan := make(chan Event[T], 10)

	id := eventbus.Subscribe(e.ID)
	if !id.IsActive() {
		close(eventChan)
		goto Token
	}

	go func() {
		for data := range id.C {
			if ev, ok := data.(Event[T]); ok {
				select {
				case eventChan <- ev:
				default:
				}
			}
		}

		close(eventChan)
	}()

Token:
	return Subscriber[T]{C: eventChan, Subscribable: id.IsActive(), Unsubscribe: id.Unsubscribe}
}

// AdapterEvent returns an event interface to publish/subscribe to adapter events.
func AdapterEvent(action ...EventAction) Event[AdapterEventData] {
	eventAction := EventActionNone
	if action != nil {
		eventAction = action[0]
	}

	return Event[AdapterEventData]{ID: EventAdapter, Action: eventAction}
}

// DeviceEvent returns an event interface to publish/subscribe to device events.
func DeviceEvent(action ...EventAction) Event[DeviceEventData] {
	eventAction := EventActionNone
	if action != nil {
		eventAction = action[0]
	}

	return Event[DeviceEventData]{ID: EventDevice, Action: eventAction}
}

// MediaEvent returns an event interface to publish/subscribe to media events.
func MediaEvent(action ...EventAction) Event[MediaEventData] {
	eventAction := EventActionNone
	if action != nil {
		eventAction = action[0]
	}

	return Event[MediaEventData]{ID: EventMediaPlayer, Action: eventAction}
}

// FileTransferEvent returns an event interface to publish/subscribe to file transfer events.
func FileTransferEvent(action ...EventAction) Event[FileTransferEventData] {
	eventAction := EventActionNone
	if action != nil {
		eventAction = action[0]
	}

	return Event[FileTransferEventData]{ID: EventFileTransfer, Action: eventAction}
}

// ErrorEvent returns an event interface to publish/subscribe to error events.
func ErrorEvent(err ...error) Event[errorkinds.GenericError] {
	ev := Event[errorkinds.GenericError]{ID: EventError, Action: EventActionAdded}
	if err != nil {
		ev.Data = errorkinds.GenericError{Errors: err[0]}
	}

	return ev
}
