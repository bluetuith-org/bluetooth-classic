package bluetooth

import (
	"github.com/bluetuith-org/bluetooth-classic/api/errorkinds"
	"github.com/bluetuith-org/bluetooth-classic/api/eventbus"
)

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

// Events defines a set of possible event data types.
type Events interface {
	NewDataEvents | UpdatedDataEvents
}

type NewDataEvents interface {
	errorkinds.GenericError | AdapterData | DeviceData | FileTransferData | MediaData
}

type UpdatedDataEvents interface {
	struct{} | AdapterEventData | DeviceEventData | FileTransferEventData | MediaEventData
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

type EventGroup[N NewDataEvents, U UpdatedDataEvents] struct {
	// ID holds the event ID.
	ID EventID
}

type Subscriber[N NewDataEvents, U UpdatedDataEvents] struct {
	AddedEvents                  chan N
	UpdatedEvents, RemovedEvents chan U
	Done                         chan struct{}

	Unsubscribe eventbus.UnsubFunc
}

func (e EventGroup[N, U]) PublishAdded(data N) {
	eventbus.Publish(e.ID, Event[N]{e.ID, EventActionAdded, data})
}

func (e EventGroup[N, U]) PublishUpdated(data U) {
	eventbus.Publish(e.ID, Event[U]{e.ID, EventActionUpdated, data})
}

func (e EventGroup[N, U]) PublishRemoved(data U) {
	eventbus.Publish(e.ID, Event[U]{e.ID, EventActionRemoved, data})
}

func (e EventGroup[N, U]) Subscribe() (*Subscriber[N, U], bool) {
	id := eventbus.Subscribe(e.ID)

	sub := Subscriber[N, U]{
		AddedEvents:   make(chan N, 1),
		RemovedEvents: make(chan U, 1),
		UpdatedEvents: make(chan U, 1),
		Done:          make(chan struct{}, 1),
		Unsubscribe:   id.Unsubscribe,
	}

	if !id.IsActive() {
		close(sub.AddedEvents)
		close(sub.RemovedEvents)
		close(sub.UpdatedEvents)
		goto Token
	}

	go func() {
		for data := range id.C {
			switch v := data.(type) {
			case Event[N]:
				if v.Action != EventActionAdded {
					continue
				}

				select {
				case sub.AddedEvents <- v.Data:
				default:
				}

			case Event[U]:
				var ch chan U

				switch v.Action {
				case EventActionUpdated:
					ch = sub.UpdatedEvents

				case EventActionRemoved:
					ch = sub.RemovedEvents

				default:
					continue
				}

				select {
				case ch <- v.Data:
				default:
				}
			}
		}

		select {
		case sub.Done <- struct{}{}:
		default:
		}

		close(sub.AddedEvents)
		close(sub.RemovedEvents)
		close(sub.UpdatedEvents)
	}()

Token:
	return &sub, id.IsActive()
}

// AdapterEvent returns an event interface to subscribe to adapter events.
func AdapterEvents(action ...EventAction) EventGroup[AdapterData, AdapterEventData] {
	return EventGroup[AdapterData, AdapterEventData]{ID: EventAdapter}
}

// DeviceEvent returns an event interface to subscribe to device events.
func DeviceEvents(action ...EventAction) EventGroup[DeviceData, DeviceEventData] {
	return EventGroup[DeviceData, DeviceEventData]{ID: EventDevice}
}

// MediaEvent returns an event interface to subscribe to media events.
func MediaEvents(action ...EventAction) EventGroup[MediaData, MediaEventData] {
	return EventGroup[MediaData, MediaEventData]{ID: EventMediaPlayer}
}

// FileTransferEvent returns an event interface to subscribe to file transfer events.
func FileTransferEvents(action ...EventAction) EventGroup[FileTransferData, FileTransferEventData] {
	return EventGroup[FileTransferData, FileTransferEventData]{ID: EventFileTransfer}
}

// ErrorEvent returns an event interface to subscribe to error events.
func ErrorEvents(err ...error) EventGroup[errorkinds.GenericError, struct{}] {
	return EventGroup[errorkinds.GenericError, struct{}]{ID: EventError}
}
