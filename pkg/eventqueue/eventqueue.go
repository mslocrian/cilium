// Copyright 2019 Authors of Cilium
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eventqueue

import (
	"fmt"
	"sync"
)

// EventQueue is a structured which is utilized to handle events for a given
// Endpoint in a generic way.
type EventQueue struct {
	// This should always be a buffered channel.
	events         chan *Event
	close          chan struct{}
	drained        chan struct{}
	eventQueueOnce sync.Once
}

func NewEventQueue() *EventQueue {
	return &EventQueue{
		// Only one event can be consumed per endpoint
		// at a time.
		events: make(chan *Event, 1),
		close:  make(chan struct{}),
		drained: make(chan struct{}),
	}

}

// Event is an event that can be queued for an Endpoint on its
// EventQueue.
type Event struct {
	// Metadata is the information about the event which is sent
	// by its queuer.
	Metadata interface{}

	// EventResults is a channel on which the results of the event are sent.
	// It is populated by the EventQueue itself, not by the queuer.
	EventResults chan interface{}

	// Cancelled is a channel which is called when the EventQueue is being drained.
	// The event was not ran if it was signaled upon.
	Cancelled chan struct{}
}

// NewEvent returns an Event with all fields initialized.
func NewEvent(meta interface{}) *Event {
	return &Event{
		Metadata:     meta,
		EventResults: make(chan interface{}, 1),
		Cancelled:    make(chan struct{}),
	}
}

func (q *Event) WasCancelled() bool {
	select {
	case <-q.Cancelled:
		return true
	default:
		return false
	}
}

func (q *EventQueue) QueueEvent(epEvent *Event) {
	select {
	case <-q.close:
		fmt.Printf("QueueEvent close epEvent.Cancelled\n")
		close(epEvent.Cancelled)
	default:
		q.events <- epEvent
	}
}

// runEventQueue consumes events that have been queued for this EventQueue. It
// is presumed that the eventQueue is a buffered channel with a length of one
// (i.e., only one event can be processed at a time).
// All business logic for handling queued events is contained within this
// function. Each event must be handled in such a way such that a result is sent
// across  its EventResults channel, as the queuer of an event may be waiting on
// a result from the event. Otherwise, if the event queue is closed, then all
// events which were queued up are cancelled. It is assumed that the caller
// handles both cases (cancel, or result) gracefully.
func (q *EventQueue) RunEventQueue() {
	q.eventQueueOnce.Do(func() {
		for {
			select {
			// Receive next event.
			case e := <-q.events:
				{
					fmt.Printf("Received event\n")
					// Handle each event type.
					switch t := e.Metadata.(type) {
					case EventHandler:
						ev := e.Metadata.(EventHandler)
						evRes := ev.Handle()
						e.EventResults <- evRes
					default:
						log.Errorf("unsupported function type provided to event queue: %T", t)
					}

					// Ensures that no more results can be sent as the event has
					// already been processed.
					close(e.EventResults)
				}
			// Cancel all events that were not yet consumed.
			case <-q.close:
				{
					fmt.Printf("closing event queue!!!\n")
					log.Debug("closing event queue")

					// Drain queue of all events. This ensures that all events that
					// nothing blocks on an EventResult which will never be created.
					for drainEvent := range q.events {
						fmt.Printf("draining queue of event\n")
						close(drainEvent.Cancelled)
					}
					fmt.Printf("closing q.drained!\n")
					close(q.drained)
					close(q.events)
					return
				}
			}
		}
	})
}

func (q *EventQueue) CloseEventQueue() {
	select {
	case <-q.close:
		log.Warning("tried to close event queue, but it already has been closed")
	default:
		log.Debug("closing event queue")
		close(q.close)
	}
}

type EventHandler interface {
	Handle() interface{}
}
