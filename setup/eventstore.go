package setup

import (
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventstore/memory"
	"github.com/looplab/eventhorizon/eventstore/mongodb"
)

func (b *Builder) WithMemoryEventStore() *Builder {
	b.buildEventStore = func() error {
		var err error
		b.setup.EventStore, err = memory.NewEventStore()
		if err != nil {
			return err
		}

		return nil
	}

	return b
}

func (b *Builder) WithMongoDBEventStore(uri, db string, options ...mongodb.Option) *Builder {
	b.buildEventStore = func() error {
		var err error
		b.setup.EventStore, err = mongodb.NewEventStore(uri, db, options...)
		if err != nil {
			return err
		}

		return nil
	}

	return b
}

func (b *Builder) WithCustomEventStore(new func() (eh.EventStore, error)) *Builder {
	b.buildEventStore = func() error {
		var err error
		b.setup.EventStore, err = new()
		if err != nil {
			return err
		}

		return nil
	}

	return b
}
