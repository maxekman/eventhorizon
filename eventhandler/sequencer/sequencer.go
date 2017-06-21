// Copyright (c) 2017 - Max Ekman <max@looplab.se>
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

package sequencer

import (
	"context"
	"errors"

	eh "github.com/looplab/eventhorizon"
	"golang.org/x/net/context"
)

// Error is an error in the projector, with the namespace.
type Error struct {
	// Err is the error.
	Err error
	// BaseErr is an optional underlying error, for example from the DB driver.
	BaseErr error
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e Error) Error() string {
	errStr := e.Err.Error()
	if e.BaseErr != nil {
		errStr += ": " + e.BaseErr.Error()
	}
	return "queue: " + errStr + " (" + e.Namespace + ")"
}

// EventHandler is an event queue that will queue events for each aggregate ID
// and let a wraped handler handle the events one by one and in order.
type EventHandler struct {
	eh.EventHandler
	ch chan eventData
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(handler eh.EventHandler) *EventHandler {
	h := &EventHandler{
		EventHandler: handler,
		ch: make(chan eventData),
	}
	go h.handle()
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	h.ch <- eventData{ctx, event}
	return nil
}

// HandlerType implements the HandlerType method of the EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return h.EventHandler.HandlerType() + eh.EventHandlerType("-sequencer")
}

type eventData struct {
	ctx context.Context
	event eh.Event
}

// Sequence events with a compond key of namespace and ID.
type key struct {
	ns string
	id eh.UUID
}

type byVersion []eh.Event
func (e byVersion) Len() int           { return len(e) }
func (e byVersion) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e byVersion) Less(i, j int) bool { return e[i].Version() < e[j].Version() }

func (h *EventHandler) handle() {
	queues := map[key][]eh.Event{}
	for {
		d, ok := <-h.ch
		if !ok {
			// Channel was closed.
			return
		}

		// Store and sort the event by version.
		ns := NamespaceFromContext(d.ctx)
		k := key{ns, d.event.AggregateID()}
		queues[k] = append(queues[k], d.event)
		sort.Sort(byVesion(queues[k]))

		err := h.EventHandler.HandleEvent(d.ctx, d.event)
		if err != nil {
			// TODO: Handle error.
		}
	}
}
