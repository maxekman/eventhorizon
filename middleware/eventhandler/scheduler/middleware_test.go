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

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

func TestCommandHandler(t *testing.T) {
	inner := mocks.NewEventHandler("test")

	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	m, scheduler := NewMiddleware(schedulerCtx)
	h := eh.UseEventHandlerMiddleware(inner, m)

	// Add the scheduler middleware to another handler to duplicate events.
	inner2 := mocks.NewEventHandler("test")
	eh.UseEventHandlerMiddleware(inner2, m)

	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	expectedEvent := eh.NewEventForAggregate(mocks.EventType, nil, timestamp,
		mocks.AggregateType, uuid.New(), 1)

	// Non-scheduled handling.
	if err := h.HandleEvent(context.Background(), expectedEvent); err != nil {
		t.Error("there should be no error:", err)
	}
	expected := []eh.Event{expectedEvent}
	if !cmp.Equal(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	inner.Events = nil

	// Schedule the same event every second.
	scheduleCtx, cancelScheduleEvent := context.WithCancel(context.Background())
	if err := scheduler.ScheduleEvent(scheduleCtx, "* * * * * * *", func(t time.Time) eh.Event {
		return expectedEvent
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	// First.
	<-time.After(time.Second)
	inner.RLock()
	expected = []eh.Event{expectedEvent}
	if !cmp.Equal(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	inner.RUnlock()

	// Second.
	<-time.After(time.Second)
	inner.RLock()
	expected = []eh.Event{expectedEvent, expectedEvent}
	if !cmp.Equal(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	inner.RUnlock()

	// Cancel before the third.
	cancelScheduleEvent()
	<-time.After(time.Second)
	inner.RLock()
	expected = []eh.Event{expectedEvent, expectedEvent}
	if !cmp.Equal(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expected, inner.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	inner.RUnlock()

	inner2.RLock()
	expected = []eh.Event{expectedEvent, expectedEvent}
	if !cmp.Equal(expected, inner2.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expected, inner2.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	inner2.RUnlock()

	// Schedule after canceled.
	cancelScheduler()
	if err := scheduler.ScheduleEvent(context.Background(), "* * * * * * *", func(t time.Time) eh.Event {
		return nil
	}); err != context.Canceled {
		t.Error("there should be a context canceled error:", err)
	}
}
