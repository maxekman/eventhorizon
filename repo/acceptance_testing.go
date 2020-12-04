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

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
)

// AcceptanceTest is the acceptance test that all implementations of Repo
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestRepo(t *testing.T) {
//       ctx := context.Background() // Or other when testing namespaces.
//       store := NewRepo()
//       repo.AcceptanceTest(t, ctx, store)
//   }
//
func AcceptanceTest(t *testing.T, ctx context.Context, repo eh.ReadWriteRepo) {
	// Find non-existing item.
	entity, err := repo.Find(ctx, uuid.New())
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityNotFound {
		t.Error("there should be a ErrEntityNotFound error:", err)
	}
	if entity != nil {
		t.Error("there should be no entity:", entity)
	}

	// FindAll with no items.
	result, err := repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 0 {
		t.Error("there should be no items:", len(result))
	}

	// Save model without ID.
	entityMissingID := &mocks.Model{
		Content:   "entity1",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	err = repo.Save(ctx, entityMissingID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.BaseErr != eh.ErrMissingEntityID {
		t.Error("there should be a ErrMissingEntityID error:", err)
	}

	// Save and find one item.
	entity1 := &mocks.Model{
		ID:        uuid.New(),
		Content:   "entity1",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	if err = repo.Save(ctx, entity1); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity1.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	expected := entity1
	if !cmp.Equal(expected, entity) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, entity))
	}

	// FindAll with one item.
	result, err = repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 1 {
		t.Error("there should be one item:", len(result))
	}
	expectedItems := []eh.Entity{entity1}
	if !cmp.Equal(expectedItems, result) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expectedItems, result))
	}

	// Save and overwrite with same ID.
	entity1Alt := &mocks.Model{
		ID:        entity1.ID,
		Content:   "entity1Alt",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	if err = repo.Save(ctx, entity1Alt); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity1Alt.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	expected = entity1Alt
	if !cmp.Equal(expected, entity) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, entity))
	}

	// Save with another ID.
	entity2 := &mocks.Model{
		ID:        uuid.New(),
		Content:   "entity2",
		CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}
	if err = repo.Save(ctx, entity2); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity2.ID)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	expected = entity2
	if !cmp.Equal(expected, entity) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, entity))
	}

	// FindAll with two items, order should be preserved from insert.
	result, err = repo.FindAll(ctx)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if len(result) != 2 {
		t.Error("there should be two items:", len(result))
	}
	// Retrieval in any order is accepted.
	expected1 := []eh.Entity{entity1Alt, entity2}
	expected2 := []eh.Entity{entity2, entity1Alt}
	if !cmp.Equal(expected1, result) && !cmp.Equal(expected2, result) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected1, result))
		t.Log(cmp.Diff(expected2, result))
	}

	// Remove item.
	if err := repo.Remove(ctx, entity1Alt.ID); err != nil {
		t.Error("there should be no error:", err)
	}
	entity, err = repo.Find(ctx, entity1Alt.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityNotFound {
		t.Error("there should be a ErrEntityNotFound error:", err)
	}
	if entity != nil {
		t.Error("there should be no entity:", entity)
	}

	// Remove non-existing item.
	err = repo.Remove(ctx, entity1Alt.ID)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityNotFound {
		t.Error("there should be a ErrEntityNotFound error:", err)
	}
}
