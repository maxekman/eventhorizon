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

package handler

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	gcpEventBus "github.com/looplab/eventhorizon/eventbus/gcp"
	localEventBus "github.com/looplab/eventhorizon/eventbus/local"
	"github.com/looplab/eventhorizon/eventhandler/waiter"
	memoryEventStore "github.com/looplab/eventhorizon/eventstore/memory"
	mongoEventStore "github.com/looplab/eventhorizon/eventstore/mongodb"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	"github.com/looplab/eventhorizon/repo/memory"
	"github.com/looplab/eventhorizon/repo/mongodb"
	"github.com/looplab/eventhorizon/repo/version"

	"github.com/looplab/eventhorizon/examples/todomvc/backend/domains/todo"
)

func TestStaticFiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("there should be a 200 status for /")
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestGetAll(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/api/todos/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `[]` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.ItemAdded},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(nil)
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `[{"id":"`+id.String()+`","version":2,"items":[{"id":0,"desc":"desc","completed":false}],"created_at":"`+todo.TimeNow().Format(time.RFC3339Nano)+`","updated_at":"`+todo.TimeNow().Format(time.RFC3339Nano)+`"}]` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestCreate(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	r := httptest.NewRequest("POST", "/api/todos/create",
		strings.NewReader(`{"id":"`+id.String()+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.Created},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(nil)
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	m, err := todoRepo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	list, ok := m.(*todo.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &todo.TodoList{
		ID:        id,
		Version:   1,
		Items:     []*todo.TodoItem{},
		CreatedAt: todo.TimeNow(),
		UpdatedAt: todo.TimeNow(),
	}
	if !cmp.Equal(expected, list) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, list))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestDelete(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/delete",
		strings.NewReader(`{"id":"`+id.String()+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.Deleted},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(nil)
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	_, err = todoRepo.Find(ctx, id)
	if rrErr, ok := err.(eh.RepoError); !ok || rrErr.Err != eh.ErrEntityNotFound {
		t.Error("there should be a not found error:", err)
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestAddItem(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/add_item",
		strings.NewReader(`{"id":"`+id.String()+`", "desc":"desc"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.ItemAdded},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(nil)
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	m, err := todoRepo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	list, ok := m.(*todo.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &todo.TodoList{
		ID:      id,
		Version: 2,
		Items: []*todo.TodoItem{
			{
				ID:          0,
				Description: "desc",
			},
		},
		CreatedAt: todo.TimeNow(),
		UpdatedAt: todo.TimeNow(),
	}
	if !cmp.Equal(expected, list) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, list))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestRemoveItem(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/remove_item",
		strings.NewReader(`{"id":"`+id.String()+`", "item_id":0}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.ItemRemoved},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(nil)
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	m, err := todoRepo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	list, ok := m.(*todo.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &todo.TodoList{
		ID:        id,
		Version:   3,
		Items:     []*todo.TodoItem{},
		CreatedAt: todo.TimeNow(),
		UpdatedAt: todo.TimeNow(),
	}
	if !cmp.Equal(expected, list) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, list))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestRemoveCompleted(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.CheckItem{
		ID:      id,
		ItemID:  1,
		Checked: true,
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/remove_completed",
		strings.NewReader(`{"id":"`+id.String()+`"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.ItemRemoved},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(func(e eh.Event) bool {
		return e.Version() == 5
	})
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	m, err := todoRepo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	list, ok := m.(*todo.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &todo.TodoList{
		ID:      id,
		Version: 5,
		Items: []*todo.TodoItem{
			{
				ID:          0,
				Description: "desc",
			},
		},
		CreatedAt: todo.TimeNow(),
		UpdatedAt: todo.TimeNow(),
	}
	if !cmp.Equal(expected, list) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, list))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestSetItemDesc(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/set_item_desc",
		strings.NewReader(`{"id":"`+id.String()+`", "desc":"new desc"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.ItemDescriptionSet},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(nil)
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	m, err := todoRepo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	list, ok := m.(*todo.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &todo.TodoList{
		ID:      id,
		Version: 3,
		Items: []*todo.TodoItem{
			{
				ID:          0,
				Description: "new desc",
			},
		},
		CreatedAt: todo.TimeNow(),
		UpdatedAt: todo.TimeNow(),
	}
	if !cmp.Equal(expected, list) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, list))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestCheckItem(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/check_item",
		strings.NewReader(`{"id":"`+id.String()+`", "item_id":1, "checked":true}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.ItemChecked},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(nil)
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	m, err := todoRepo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	list, ok := m.(*todo.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &todo.TodoList{
		ID:      id,
		Version: 4,
		Items: []*todo.TodoItem{
			{
				ID:          0,
				Description: "desc",
			},
			{
				ID:          1,
				Description: "completed",
				Completed:   true,
			},
		},
		CreatedAt: todo.TimeNow(),
		UpdatedAt: todo.TimeNow(),
	}
	if !cmp.Equal(expected, list) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, list))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func TestCheckAllItems(t *testing.T) {
	todo.TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	commandHandler, eventBus, todoRepo := NewTestSession(ctx)

	h, err := NewHandler(ctx, commandHandler, eventBus, todoRepo, "../../frontend")
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err := commandHandler.HandleCommand(ctx, &todo.Create{
		ID: id,
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "desc",
	}); err != nil {
		t.Error("there should be no error:", err)
	}
	if err := commandHandler.HandleCommand(ctx, &todo.AddItem{
		ID:          id,
		Description: "completed",
	}); err != nil {
		t.Error("there should be no error:", err)
	}

	r := httptest.NewRequest("POST", "/api/todos/check_all_items",
		strings.NewReader(`{"id":"`+id.String()+`", "item_id":1, "checked":true}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Error("the status should be correct:", w.Code)
	}
	if string(w.Body.Bytes()) != `` {
		t.Error("the body should be correct:", string(w.Body.Bytes()))
	}

	waiter := waiter.NewEventHandler()
	eventBus.AddHandler(ctx, eh.MatchEvents{todo.ItemRemoved},
		eh.UseEventHandlerMiddleware(waiter, observer.Middleware))
	l := waiter.Listen(func(e eh.Event) bool {
		return e.Version() == 5
	})
	var cancelTimeout func()
	ctx, cancelTimeout = context.WithTimeout(ctx, time.Second)
	l.Wait(ctx)
	cancelTimeout()

	m, err := todoRepo.Find(ctx, id)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	list, ok := m.(*todo.TodoList)
	if !ok {
		t.Error("the item should be a todo list")
	}
	expected := &todo.TodoList{
		ID:      id,
		Version: 5,
		Items: []*todo.TodoItem{
			{
				ID:          0,
				Description: "desc",
				Completed:   true,
			},
			{
				ID:          1,
				Description: "completed",
				Completed:   true,
			},
		},
		CreatedAt: todo.TimeNow(),
		UpdatedAt: todo.TimeNow(),
	}
	if !cmp.Equal(expected, list) {
		t.Error("the item should be correct:")
		t.Log(cmp.Diff(expected, list))
	}

	// Cancel all handlers and wait.
	cancel()
	eventBus.Wait()
}

func NewTestSession(ctx context.Context) (
	eh.CommandHandler,
	eh.EventBus,
	eh.ReadWriteRepo,
) {
	commandBus := bus.NewCommandHandler()
	eventStore := memoryEventStore.NewEventStore()
	eventBus := localEventBus.NewEventBus(nil)
	todoRepo := memory.NewRepo()
	if err := todo.SetupDomain(ctx, commandBus, eventStore, eventBus, todoRepo); err != nil {
		log.Println("could not setup domain:", err)
	}
	return commandBus, eventBus, todoRepo
}

func NewIntegrationTestSession(ctx context.Context) (
	eh.CommandHandler,
	eh.EventBus,
	eh.ReadWriteRepo,
) {
	// Use MongoDB in Docker with fallback to localhost.
	dbURL := os.Getenv("MONGO_HOST")
	if dbURL == "" {
		dbURL = "localhost:27017"
	}
	dbURL = "mongodb://" + dbURL
	dbPrefix := "todomvc-example"

	commandBus := bus.NewCommandHandler()

	eventStore, err := mongoEventStore.NewEventStore(dbURL, dbPrefix)
	if err != nil {
		log.Fatalf("could not create event store: %s", err)
	}

	eventBus, err := gcpEventBus.NewEventBus("project-id", dbPrefix)
	if err != nil {
		log.Fatalf("could not create event bus: %s", err)
	}
	go func() {
		for e := range eventBus.Errors() {
			log.Printf("eventbus: %s", e.Error())
		}
	}()

	repo, err := mongodb.NewRepo(dbURL, dbPrefix, "todos")
	if err != nil {
		log.Fatalf("could not create invitation repository: %s", err)
	}
	todoRepo := version.NewRepo(repo)

	// NOTE: Temp clear of DB on startup.
	mongoRepo, ok := todoRepo.Parent().(*mongodb.Repo)
	if !ok {
		log.Fatal("incorrect repo type")
	}
	if err := mongoRepo.Clear(ctx); err != nil {
		log.Println("could not clear DB:", err)
	}

	if err := todo.SetupDomain(ctx, commandBus, eventStore, eventBus, todoRepo); err != nil {
		log.Println("could not setup domain:", err)
	}

	return commandBus, eventBus, todoRepo
}
