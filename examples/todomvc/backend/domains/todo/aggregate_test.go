// Copyright (c) 2017 - The Event Horizon authors.
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

package todo

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/mocks"
)

func TestAggregateHandleCommand(t *testing.T) {
	TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.Local)
	}

	id := uuid.New()
	cases := map[string]struct {
		agg            *Aggregate
		cmd            eh.Command
		expectedEvents []eh.Event
		expectedErr    error
	}{
		"unknown command": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
			},
			&mocks.Command{
				ID:      id,
				Content: "testcontent",
			},
			nil,
			mocks.ExpectedError("could not handle command: Command"),
		},
		"create": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
			},
			&Create{
				ID: id,
			},
			[]eh.Event{
				eh.NewEventForAggregate(Created, nil,
					TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
		"create (already created)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
			},
			&Create{},
			nil,
			mocks.ExpectedError("already created"),
		},
		"delete": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
			},
			&Delete{},
			[]eh.Event{
				eh.NewEventForAggregate(Deleted, nil,
					TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
		"delete (not created)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
			},
			&Delete{},
			nil,
			mocks.ExpectedError("not created"),
		},
		"add item": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				nextItemID:    1,
			},
			&AddItem{
				Description: "desc",
			},
			[]eh.Event{
				eh.NewEventForAggregate(ItemAdded, &ItemAddedData{
					ItemID:      1,
					Description: "desc",
				}, TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
		"remove item": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc",
						Completed:   true,
					},
				},
			},
			&RemoveItem{
				ItemID: 1,
			},
			[]eh.Event{
				eh.NewEventForAggregate(ItemRemoved, &ItemRemovedData{
					ItemID: 1,
				}, TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
		"remove item (non existing)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc",
						Completed:   true,
					},
				},
			},
			&RemoveItem{
				ItemID: 2,
			},
			nil,
			mocks.ExpectedError("item does not exist: 2"),
		},
		"remove completed items": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   true,
					},
					{
						ID:          2,
						Description: "desc 2",
						Completed:   false,
					},
					{
						ID:          3,
						Description: "desc 3",
						Completed:   true,
					},
				},
			},
			&RemoveCompletedItems{},
			[]eh.Event{
				eh.NewEventForAggregate(ItemRemoved, &ItemRemovedData{
					ItemID: 1,
				}, TimeNow(), AggregateType, id, 1),
				eh.NewEventForAggregate(ItemRemoved, &ItemRemovedData{
					ItemID: 3,
				}, TimeNow(), AggregateType, id, 2),
			},
			nil,
		},
		"set item description": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   true,
					},
				},
			},
			&SetItemDescription{
				ItemID:      1,
				Description: "new desc",
			},
			[]eh.Event{
				eh.NewEventForAggregate(ItemDescriptionSet, &ItemDescriptionSetData{
					ItemID:      1,
					Description: "new desc",
				}, TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
		"set item description (non existing)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   true,
					},
				},
			},
			&SetItemDescription{
				ItemID:      2,
				Description: "new desc",
			},
			nil,
			mocks.ExpectedError("item does not exist: 2"),
		},
		"set item description (no change)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   true,
					},
				},
			},
			&SetItemDescription{
				ItemID:      1,
				Description: "desc 1",
			},
			nil,
			nil,
		},
		"check item": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
				},
			},
			&CheckItem{
				ItemID:  1,
				Checked: true,
			},
			[]eh.Event{
				eh.NewEventForAggregate(ItemChecked, &ItemCheckedData{
					ItemID:  1,
					Checked: true,
				}, TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
		"uncheck item": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   true,
					},
				},
			},
			&CheckItem{
				ItemID:  1,
				Checked: false,
			},
			[]eh.Event{
				eh.NewEventForAggregate(ItemChecked, &ItemCheckedData{
					ItemID:  1,
					Checked: false,
				}, TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
		"check item (non exsisting)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
				},
			},
			&CheckItem{
				ItemID:  2,
				Checked: true,
			},
			nil,
			mocks.ExpectedError("item does not exist: 2"),
		},
		"check item (no change)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   true,
					},
				},
			},
			&CheckItem{
				ItemID:  1,
				Checked: true,
			},
			nil,
			nil,
		},
		"check all items": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
					{
						ID:          2,
						Description: "desc 2",
						Completed:   true,
					},
					{
						ID:          3,
						Description: "desc 3",
						Completed:   false,
					},
				},
			},
			&CheckAllItems{
				Checked: true,
			},
			[]eh.Event{
				eh.NewEventForAggregate(ItemChecked, &ItemCheckedData{
					ItemID:  1,
					Checked: true,
				}, TimeNow(), AggregateType, id, 1),
				eh.NewEventForAggregate(ItemChecked, &ItemCheckedData{
					ItemID:  3,
					Checked: true,
				}, TimeNow(), AggregateType, id, 2),
			},
			nil,
		},
		"uncheck all items": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
					{
						ID:          2,
						Description: "desc 2",
						Completed:   true,
					},
					{
						ID:          3,
						Description: "desc 3",
						Completed:   false,
					},
				},
			},
			&CheckAllItems{
				Checked: false,
			},
			[]eh.Event{
				eh.NewEventForAggregate(ItemChecked, &ItemCheckedData{
					ItemID:  2,
					Checked: false,
				}, TimeNow(), AggregateType, id, 1),
			},
			nil,
		},
	}

	for name, tc := range cases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := tc.agg.HandleCommand(context.Background(), tc.cmd)
			if !cmp.Equal(tc.expectedErr, err, cmpopts.EquateErrors()) {
				t.Errorf("test case '%s': incorrect error", name)
				t.Log(cmp.Diff(tc.expectedErr, err, cmpopts.EquateErrors()))
			}
			events := tc.agg.Events()
			if !cmp.Equal(tc.expectedEvents, events, cmp.AllowUnexported(eh.EmptyEvent)) {
				t.Errorf("test case '%s': incorrect events", name)
				t.Log(cmp.Diff(tc.expectedEvents, events, cmp.AllowUnexported(eh.EmptyEvent)))
			}
		})
	}
}

func TestAggregateApplyEvent(t *testing.T) {
	TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.Local)
	}

	id := uuid.New()
	cases := map[string]struct {
		agg         *Aggregate
		event       eh.Event
		expectedAgg *Aggregate
		expectedErr error
	}{
		"unhandeled event": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
			},
			eh.NewEventForAggregate(eh.EventType("unknown"), nil,
				TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
			},
			mocks.ExpectedError("could not apply event: unknown"),
		},
		"created": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
			},
			eh.NewEventForAggregate(Created, nil,
				TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
			},
			nil,
		},
		"deleted": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				created:       true,
			},
			eh.NewEventForAggregate(Deleted, nil,
				TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
			},
			nil,
		},
		"item added": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				nextItemID:    1,
			},
			eh.NewEventForAggregate(ItemAdded, &ItemAddedData{
				ItemID:      1,
				Description: "desc 1",
			}, TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				nextItemID:    2,
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
				},
			},
			nil,
		},
		"item removed": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
					{
						ID:          2,
						Description: "desc 2",
						Completed:   false,
					},
				},
			},
			eh.NewEventForAggregate(ItemRemoved, &ItemRemovedData{
				ItemID: 2,
			}, TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
				},
			},
			nil,
		},
		"item removed (last)": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
				},
			},
			eh.NewEventForAggregate(ItemRemoved, &ItemRemovedData{
				ItemID: 1,
			}, TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items:         []*TodoItem{},
			},
			nil,
		},
		"item description set": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
					{
						ID:          2,
						Description: "desc 2",
						Completed:   false,
					},
				},
			},
			eh.NewEventForAggregate(ItemDescriptionSet, &ItemDescriptionSetData{
				ItemID:      2,
				Description: "new desc",
			}, TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
					{
						ID:          2,
						Description: "new desc",
						Completed:   false,
					},
				},
			},
			nil,
		},
		"item checked": {
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
					{
						ID:          2,
						Description: "desc 2",
						Completed:   false,
					},
				},
			},
			eh.NewEventForAggregate(ItemChecked, &ItemCheckedData{
				ItemID:  2,
				Checked: true,
			}, TimeNow(), AggregateType, id, 1),
			&Aggregate{
				AggregateBase: events.NewAggregateBase(AggregateType, id),
				items: []*TodoItem{
					{
						ID:          1,
						Description: "desc 1",
						Completed:   false,
					},
					{
						ID:          2,
						Description: "desc 2",
						Completed:   true,
					},
				},
			},
			nil,
		},
	}

	for name, tc := range cases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := tc.agg.ApplyEvent(context.Background(), tc.event)
			if !cmp.Equal(tc.expectedErr, err, cmpopts.EquateErrors()) {
				t.Errorf("test case '%s': incorrect error", name)
				t.Log(cmp.Diff(tc.expectedErr, err, cmpopts.EquateErrors()))
			}
			if !cmp.Equal(tc.expectedAgg, tc.agg, cmp.AllowUnexported(events.AggregateBase{}, Aggregate{})) {
				t.Errorf("test case '%s': incorrect aggregate", name)
				t.Log(cmp.Diff(tc.expectedAgg, tc.agg, cmp.AllowUnexported(events.AggregateBase{}, Aggregate{})))
			}
		})
	}
}
