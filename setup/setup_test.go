package setup

import "testing"

func TestSetup(t *testing.T) {
	s, err := New().
		WithCommandBus().
		WithMemoryEventStore().
		WithLocalEventBus().
		Build()
	if err != nil {
		t.Fatal(err)
	}

	if s.CommandBus == nil {
		t.Error("there should be a command bus")
	}
	if s.EventStore == nil {
		t.Error("there should be a event store")
	}
	if s.EventBus == nil {
		t.Error("there should be a event bus")
	}
}
