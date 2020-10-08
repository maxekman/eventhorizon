// Copyright (c) 2020 - The Event Horizon authors.
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

package multi

import (
	"context"
	"strings"

	eh "github.com/looplab/eventhorizon"
)

// EventHandler is a CQRS projection handler to run a Projector implementation.
type EventHandler struct {
	handlers []eh.EventHandler
}

var _ = eh.EventHandler(&EventHandler{})

// NewEventHandler creates a new EventHandler.
func NewEventHandler(handlers ...eh.EventHandler) *EventHandler {
	return &EventHandler{
		handlers: handlers,
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	hts := []string{}
	for _, h := range h.handlers {
		hts = append(hts, string(h.HandlerType()))
	}
	return eh.EventHandlerType("multi_" + strings.Join(hts, "_"))
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
// It will handle the event with all handlers in order and stop on and return or
// the first error.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	for _, h := range h.handlers {
		return h.HandleEvent(ctx, event)
	}
	return nil
}
