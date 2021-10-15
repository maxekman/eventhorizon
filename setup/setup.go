package setup

import (
	"fmt"

	eh "github.com/looplab/eventhorizon"
)

// Setup is a complete setup for a Event Horizon service.
type Setup struct {
	CommandBus eh.CommandHandler
	EventStore eh.EventStore
	EventBus   eh.EventBus
}

// Builder is a builder of a Event Horizon setup.
type Builder struct {
	setup *Setup

	buildCommandBus func() error
	buildEventStore func() error
	buildEventBus   func() error
}

// New creates a new Builder which should be populated with configuration
// options and then built by calling Build():
//
//   s, err := setup.New().
//       WithCommandBus().
//       WithMongoDBEventStore(...).
//       WithRedisEventBus(...).
//       Build()
//   if err != nil {
//       log.Fatal(err)
//   }
func New() *Builder {
	return &Builder{
		setup: &Setup{},
	}
}

// Build builds a Setup from its configuration.
func (b *Builder) Build() (*Setup, error) {
	if err := b.buildCommandBus(); err != nil {
		return nil, fmt.Errorf("could not setup command bus: %w", err)
	}

	if err := b.buildEventStore(); err != nil {
		return nil, fmt.Errorf("could not setup event store: %w", err)
	}

	if err := b.buildEventBus(); err != nil {
		return nil, fmt.Errorf("could not setup event bus: %w", err)
	}

	return b.setup, nil
}
