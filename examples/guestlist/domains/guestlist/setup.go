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

package guestlist

import (
	"context"
	"log"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
)

// Setup configures the guestlist.
func Setup(
	ctx context.Context,
	eventStore eh.EventStore,
	eventBus eh.EventBus,
	commandBus *bus.CommandHandler,
	invitationRepo, guestListRepo eh.ReadWriteRepo,
	eventID uuid.UUID) {

	// Add a logger as an observer.
	eventBus.AddHandler(ctx, eh.MatchAll{},
		eh.UseEventHandlerMiddleware(&Logger{}, observer.Middleware))

	// Create the aggregate repository.
	aggregateStore, err := events.NewAggregateStore(eventStore)
	if err != nil {
		log.Fatalf("could not create aggregate store: %s", err)
	}

	// Create the aggregate command handler and register the commands it handles.
	invitationHandler, err := aggregate.NewCommandHandler(InvitationAggregateType, aggregateStore)
	if err != nil {
		log.Fatalf("could not create command handler: %s", err)
	}
	commandHandler := eh.UseCommandHandlerMiddleware(invitationHandler, LoggingMiddleware)
	commandBus.SetHandler(commandHandler, CreateInviteCommand)
	commandBus.SetHandler(commandHandler, AcceptInviteCommand)
	commandBus.SetHandler(commandHandler, DeclineInviteCommand)
	commandBus.SetHandler(commandHandler, ConfirmInviteCommand)
	commandBus.SetHandler(commandHandler, DenyInviteCommand)

	// Create and register a read model for individual invitations.
	invitationProjector := projector.NewEventHandler(
		NewInvitationProjector(), invitationRepo)
	invitationProjector.SetEntityFactory(func() eh.Entity { return &Invitation{} })
	eventBus.AddHandler(ctx, eh.MatchEvents{
		InviteCreatedEvent,
		InviteAcceptedEvent,
		InviteDeclinedEvent,
		InviteConfirmedEvent,
		InviteDeniedEvent,
	}, invitationProjector)

	// Create and register a read model for a guest list.
	guestListProjector := NewGuestListProjector(guestListRepo, eventID)
	eventBus.AddHandler(ctx, eh.MatchEvents{
		InviteAcceptedEvent,
		InviteDeclinedEvent,
		InviteConfirmedEvent,
		InviteDeniedEvent,
	}, guestListProjector)

	// Setup the saga that responds to the accepted guests and limits the total
	// amount of guests, responding with a confirmation or denial.
	responseSaga := saga.NewEventHandler(NewResponseSaga(2), commandBus)
	eventBus.AddHandler(ctx, eh.MatchEvents{InviteAcceptedEvent}, responseSaga)
}
