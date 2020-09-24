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

package factory

import (
	"context"
	"testing"

	eh "github.com/looplab/eventhorizon"
	ehid "github.com/looplab/eventhorizon/id/google_uuid"
)

func init() {
	ehid.UseAsIDType()
}

func TestCreateAggregate(t *testing.T) {
	id := ehid.NewID()
	aggregate, err := CreateAggregate(TestAggregateRegisterType, id)
	if err != ErrAggregateNotRegistered {
		t.Error("there should be a aggregate not registered error:", err)
	}

	RegisterAggregate(func(id eh.ID) eh.Aggregate {
		if id, ok := id.(ehid.ID); ok {
			return &TestAggregateRegister{id: id}
		}
		return nil
	})

	aggregate, err = CreateAggregate(TestAggregateRegisterType, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	// NOTE: The aggregate type used to register with is another than the aggregate!
	if aggregate.AggregateType() != TestAggregateRegisterType {
		t.Error("the aggregate type should be correct:", aggregate.AggregateType())
	}
	if aggregate.EntityID() != id {
		t.Error("the ID should be correct:", aggregate.EntityID())
	}
}

func TestRegisterAggregateEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: attempt to register empty aggregate type" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id eh.ID) eh.Aggregate {
		if id, ok := id.(ehid.ID); ok {
			return &TestAggregateRegisterEmpty{id: id}
		}
		return nil
	})
}

func TestRegisterAggregateNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: created aggregate is nil" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id eh.ID) eh.Aggregate { return nil })
}

func TestRegisterAggregateTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || r != "eventhorizon: registering duplicate types for \"TestAggregateRegisterTwice\"" {
			t.Error("there should have been a panic:", r)
		}
	}()
	RegisterAggregate(func(id eh.ID) eh.Aggregate {
		if id, ok := id.(ehid.ID); ok {
			return &TestAggregateRegisterTwice{id: id}
		}
		return nil
	})
	RegisterAggregate(func(id eh.ID) eh.Aggregate {
		if id, ok := id.(ehid.ID); ok {
			return &TestAggregateRegisterTwice{id: id}
		}
		return nil
	})
}

const (
	TestAggregateRegisterType      eh.AggregateType = "TestAggregateRegister"
	TestAggregateRegisterEmptyType eh.AggregateType = ""
	TestAggregateRegisterTwiceType eh.AggregateType = "TestAggregateRegisterTwice"
)

type TestAggregateRegister struct {
	id ehid.ID
}

var _ = eh.Aggregate(&TestAggregateRegister{})

func (a *TestAggregateRegister) EntityID() eh.ID { return a.id }

func (a *TestAggregateRegister) AggregateType() eh.AggregateType {
	return TestAggregateRegisterType
}
func (a *TestAggregateRegister) HandleCommand(ctx context.Context, cmd eh.Command) error {
	return nil
}

type TestAggregateRegisterEmpty struct {
	id ehid.ID
}

var _ = eh.Aggregate(&TestAggregateRegisterEmpty{})

func (a *TestAggregateRegisterEmpty) EntityID() eh.ID { return a.id }

func (a *TestAggregateRegisterEmpty) AggregateType() eh.AggregateType {
	return TestAggregateRegisterEmptyType
}
func (a *TestAggregateRegisterEmpty) HandleCommand(ctx context.Context, cmd eh.Command) error {
	return nil
}

type TestAggregateRegisterTwice struct {
	id ehid.ID
}

var _ = eh.Aggregate(&TestAggregateRegisterTwice{})

func (a *TestAggregateRegisterTwice) EntityID() eh.ID { return a.id }

func (a *TestAggregateRegisterTwice) AggregateType() eh.AggregateType {
	return TestAggregateRegisterTwiceType
}
func (a *TestAggregateRegisterTwice) HandleCommand(ctx context.Context, cmd eh.Command) error {
	return nil
}
