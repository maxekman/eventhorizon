// Copyright (c) 2016 - The Event Horizon authors.
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

package eventbus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	"github.com/looplab/eventhorizon/mocks"
)

// AcceptanceTest is the acceptance test that all implementations of EventBus
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestEventBus(t *testing.T) {
//       bus1 := NewEventBus()
//       bus2 := NewEventBus()
//       eventbus.AcceptanceTest(t, bus1, bus2)
//   }
//
func AcceptanceTest(t *testing.T, bus1, bus2 eh.EventBus, timeout time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())

	// Error on nil matcher.
	if err := bus1.AddHandler(ctx, nil, mocks.NewEventHandler("no-matcher")); err != eh.ErrMissingMatcher {
		t.Error("the error should be correct:", err)
	}

	// Error on nil handler.
	if err := bus1.AddHandler(ctx, eh.MatchAll{}, nil); err != eh.ErrMissingHandler {
		t.Error("the error should be correct:", err)
	}

	// Error on multiple registrations.
	if err := bus1.AddHandler(ctx, eh.MatchAll{}, mocks.NewEventHandler("multi")); err != nil {
		t.Error("there should be no errer:", err)
	}
	if err := bus1.AddHandler(ctx, eh.MatchAll{}, mocks.NewEventHandler("multi")); err != eh.ErrHandlerAlreadyAdded {
		t.Error("the error should be correct:", err)
	}

	ctx = mocks.WithContextOne(ctx, "testval")

	// Without handler.
	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	event1 := eh.NewEventForAggregate(mocks.EventType, &mocks.EventData{Content: "event1"}, timestamp,
		mocks.AggregateType, id, 1)
	if err := bus1.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	const (
		handlerName  = "handler"
		observerName = "observer"
	)

	// Event without data (tested in its own handler).
	otherHandler := mocks.NewEventHandler("other-handler")
	bus1.AddHandler(ctx, eh.MatchEvents{mocks.EventOtherType}, otherHandler)
	eventWithoutData := eh.NewEventForAggregate(mocks.EventOtherType, nil, timestamp,
		mocks.AggregateType, uuid.New(), 1)
	if err := bus1.HandleEvent(ctx, eventWithoutData); err != nil {
		t.Error("there should be no error:", err)
	}
	expectedEvents := []eh.Event{eventWithoutData}
	if !otherHandler.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	if !cmp.Equal(expectedEvents, otherHandler.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expectedEvents, otherHandler.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}

	// Add handlers and observers.
	handlerBus1 := mocks.NewEventHandler(handlerName)
	handlerBus2 := mocks.NewEventHandler(handlerName)
	anotherHandlerBus2 := mocks.NewEventHandler("another_handler")
	observerBus1 := mocks.NewEventHandler(observerName)
	observerBus2 := mocks.NewEventHandler(observerName)
	bus1.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, handlerBus1)
	bus2.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, handlerBus2)
	bus2.AddHandler(ctx, eh.MatchEvents{mocks.EventType}, anotherHandlerBus2)
	// Add observers using the observer middleware.
	bus1.AddHandler(ctx, eh.MatchAll{}, eh.UseEventHandlerMiddleware(observerBus1, observer.Middleware))
	bus2.AddHandler(ctx, eh.MatchAll{}, eh.UseEventHandlerMiddleware(observerBus2, observer.Middleware))

	// Event with data.
	if err := bus1.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	// Check for correct event in handler 1 or 2.
	expectedEvents = []eh.Event{event1}
	if !(handlerBus1.Wait(timeout) || handlerBus2.Wait(timeout)) {
		t.Error("did not receive event in time")
	}
	if !(cmp.Equal(expectedEvents, handlerBus1.Events, cmp.AllowUnexported(eh.EmptyEvent)) ||
		cmp.Equal(expectedEvents, handlerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent))) {
		t.Error("the events should be correct correct in ONE of the handlers on the same bus:")
		t.Log(cmp.Diff(expectedEvents, handlerBus1.Events, cmp.AllowUnexported(eh.EmptyEvent)))
		t.Log(cmp.Diff(expectedEvents, handlerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	if cmp.Equal(handlerBus1.Events, handlerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("only one handler should receive the events:")
		t.Log(cmp.Diff(handlerBus1.Events, handlerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	correctCtx1 := false
	if val, ok := mocks.ContextOne(handlerBus1.Context); ok && val == "testval" {
		correctCtx1 = true
	}
	correctCtx2 := false
	if val, ok := mocks.ContextOne(handlerBus2.Context); ok && val == "testval" {
		correctCtx2 = true
	}
	if !correctCtx1 && !correctCtx2 {
		t.Error("the context should be correct")
	}

	// Check the other handler.
	if !anotherHandlerBus2.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	if !cmp.Equal(expectedEvents, anotherHandlerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expectedEvents, anotherHandlerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	if val, ok := mocks.ContextOne(anotherHandlerBus2.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", anotherHandlerBus2.Context)
	}

	// Check observer 1.
	if !observerBus1.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	if !cmp.Equal(expectedEvents, observerBus1.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expectedEvents, observerBus1.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	if val, ok := mocks.ContextOne(observerBus1.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observerBus1.Context)
	}

	// Check observer 2.
	if !observerBus2.Wait(timeout) {
		t.Error("did not receive event in time")
	}
	if !cmp.Equal(expectedEvents, observerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent)) {
		t.Error("the events should be correct:")
		t.Log(cmp.Diff(expectedEvents, observerBus2.Events, cmp.AllowUnexported(eh.EmptyEvent)))
	}
	if val, ok := mocks.ContextOne(observerBus2.Context); !ok || val != "testval" {
		t.Error("the context should be correct:", observerBus2.Context)
	}

	// Test async errors from handlers.
	errorHandler := mocks.NewEventHandler("error_handler")
	errorHandler.Err = errors.New("handler error")
	bus1.AddHandler(ctx, eh.MatchAll{}, errorHandler)
	if err := bus1.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}
	select {
	case <-time.After(time.Second):
		t.Error("there should be an async error")
	case err := <-bus1.Errors():
		// Good case.
		if err.Error() != "could not handle event (error_handler): handler error: (Event@1)" {
			t.Error(err, "wrong error sent on event bus")
		}
	}

	// Cancel all handlers and wait.
	cancel()
	bus1.Wait()
	bus2.Wait()
}

// LoadTest is a load test for an event bus implementation.
func LoadTest(t *testing.T, bus eh.EventBus) {
	ctx, cancel := context.WithCancel(context.Background())

	handlers := make([]*mocks.EventHandler, 100)
	var wg sync.WaitGroup
	for i := range handlers {
		h := mocks.NewEventHandler(fmt.Sprintf("handler-%d", i))
		if err := bus.AddHandler(ctx, eh.MatchAll{}, h); err != nil {
			t.Error("there should be no error:", err)
		}
		wg.Add(1)
		handlers[i] = h
		go func() {
			<-h.Recv
			wg.Done()
		}()
	}

	t.Log("setup complete")

	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	event1 := eh.NewEventForAggregate(
		mocks.EventType, &mocks.EventData{Content: "event1"},
		timestamp, mocks.AggregateType, id, 1)
	if err := bus.HandleEvent(ctx, event1); err != nil {
		t.Error("there should be no error:", err)
	}

	wg.Wait()

	// Cancel all handlers and wait.
	cancel()
	bus.Wait()
}

// Benchmark is a benchmark for an event bus implementation.
func Benchmark(b *testing.B, bus eh.EventBus) {
	ctx, cancel := context.WithCancel(context.Background())

	id := uuid.New()
	timestamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	b.Log("setup complete")
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		event1 := eh.NewEventForAggregate(
			mocks.EventType, &mocks.EventData{Content: "event1"},
			timestamp, mocks.AggregateType, id, n+1)
		if err := bus.HandleEvent(ctx, event1); err != nil {
			b.Error("there should be no error:", err)
		}
	}

	// Cancel all handlers and wait.
	cancel()
	bus.Wait()
}
