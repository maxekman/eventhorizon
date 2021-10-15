package setup

import (
	"github.com/looplab/eventhorizon/commandhandler/bus"
)

func (b *Builder) WithCommandBus() *Builder {
	b.buildCommandBus = func() error {
		b.setup.CommandBus = bus.NewCommandHandler()
		return nil
	}
	return b
}
