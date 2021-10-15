package setup

import (
	"github.com/looplab/eventhorizon/eventbus/local"
	"github.com/looplab/eventhorizon/eventbus/redis"
)

func (b *Builder) WithLocalEventBus(options ...local.Option) *Builder {
	b.buildEventBus = func() error {
		b.setup.EventBus = local.NewEventBus(options...)
		return nil
	}

	return b
}

func (b *Builder) WithRedisEventBus(addr, appID, clientID string, options ...redis.Option) *Builder {
	b.buildEventBus = func() error {
		var err error
		b.setup.EventBus, err = redis.NewEventBus(addr, appID, clientID, options...)
		if err != nil {
			return err
		}

		return nil
	}

	return b
}
