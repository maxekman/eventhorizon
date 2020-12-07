// Copyright (c) 2014 - The Event Horizon authors.
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

package mocks

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
)

// ExpectedError is an error that is equal to any error with the same string repr.
type ExpectedError string

// Error implements the error.Error method.
func (e ExpectedError) Error() string { return string(e) }

// Is implements the Is method of the errors packege.
func (e ExpectedError) Is(err error) bool { return err != nil && e.Error() == err.Error() }

// CompareEvent compares two events.
func CompareEvent(x, y eh.Event) bool {
	if x.EventType() != y.EventType() {
		return false
	}
	if x.Timestamp() != y.Timestamp() {
		return false
	}
	if x.AggregateType() != y.AggregateType() {
		return false
	}
	if x.AggregateID() != y.AggregateID() {
		return false
	}
	if x.Version() != y.Version() {
		return false
	}
	if !cmp.Equal(x.Data(), y.Data()) {
		return false
	}
	return true
}

type ComparableEvent struct {
	EventType     eh.EventType
	Data          eh.EventData
	Timestamp     time.Time
	AggregateType eh.AggregateType
	AggregateID   uuid.UUID
	Version       int
}

func EventTransformer(e eh.Event) *ComparableEvent {
	fmt.Printf("TRANSFORMING %T\n", e)
	return &ComparableEvent{
		EventType:     e.EventType(),
		Data:          e.Data(),
		Timestamp:     e.Timestamp(),
		AggregateType: e.AggregateType(),
		AggregateID:   e.AggregateID(),
		Version:       e.Version(),
	}
}
