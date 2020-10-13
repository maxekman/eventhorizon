// Copyright (c) 2015 - The Event Horizon authors
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

package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	// Register uuid.UUID as BSON type.
	_ "github.com/looplab/eventhorizon/codec/bson"

	eh "github.com/looplab/eventhorizon"
)

// ErrCouldNotDialDB is when the database could not be dialed.
var ErrCouldNotDialDB = errors.New("could not dial database")

// ErrNoDBClient is when no database client is set.
var ErrNoDBClient = errors.New("no database client")

// ErrCouldNotClearDB is when the database could not be cleared.
var ErrCouldNotClearDB = errors.New("could not clear database")

// ErrCouldNotMarshalEvent is when an event could not be marshaled into BSON.
var ErrCouldNotMarshalEvent = errors.New("could not marshal event")

// ErrCouldNotUnmarshalEvent is when an event could not be unmarshaled into a concrete type.
var ErrCouldNotUnmarshalEvent = errors.New("could not unmarshal event")

// ErrCouldNotLoadAggregate is when an aggregate could not be loaded.
var ErrCouldNotLoadAggregate = errors.New("could not load aggregate")

// ErrCouldNotStartSession is when a DB session could not be started.
var ErrCouldNotStartSession = errors.New("could not start session")

// ErrCouldNotSaveAggregate is when an aggregate could not be saved.
var ErrCouldNotSaveAggregate = errors.New("could not save aggregate")

// Outbox implements an Outbox for MongoDB.
type Outbox struct {
	client          *mongo.Client
	dbPrefix        string
	dbName          func(ctx context.Context) string
	eventHandler    eh.EventHandler
	useTransactions bool
}

// NewOutbox creates a new Outbox with a MongoDB URI: `mongodb://hostname`.
func NewOutbox(uri, dbPrefix string, options ...Option) (*Outbox, error) {
	opts := mongoOptions.Client().ApplyURI(uri)
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	opts.SetReadConcern(readconcern.Majority())
	opts.SetReadPreference(readpref.Primary())
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, ErrCouldNotDialDB
	}

	return NewOutboxWithClient(client, dbPrefix, options...)
}

// NewOutboxWithClient creates a new Outbox with a client.
func NewOutboxWithClient(client *mongo.Client, dbPrefix string, options ...Option) (*Outbox, error) {
	if client == nil {
		return nil, ErrNoDBClient
	}

	s := &Outbox{
		client:   client,
		dbPrefix: dbPrefix,
	}

	// Use the a prefix and namespcae from the context for DB name.
	s.dbName = func(ctx context.Context) string {
		ns := eh.NamespaceFromContext(ctx)
		return dbPrefix + "_" + ns
	}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, fmt.Errorf("error while applying option: %v", err)
		}
	}

	return s, nil
}

// Option is an option setter used to configure creation.
type Option func(*Outbox) error

// WithPrefixAsDBName uses only the prefix as DB name, without namespace support.
func WithPrefixAsDBName() Option {
	return func(s *Outbox) error {
		s.dbName = func(context.Context) string {
			return s.dbPrefix
		}
		return nil
	}
}

// WithDBName uses a custom DB name function.
func WithDBName(dbName func(context.Context) string) Option {
	return func(s *Outbox) error {
		s.dbName = dbName
		return nil
	}
}

func (s *Outbox) HandleEvent(ctx context.Context, event eh.Event) error {
	// TODO: Store event in outbox.

	// Build all event records, with incrementing versions starting from the
	// original aggregate version.
	eventsToInsert := make([]*event, len(events))
	// A slice of events that can be directly inserted in the DB.
	dbEvents := make([]*dbEvent, len(events))
	aggregateID := events[0].AggregateID()
	version := originalVersion
	for i, event := range events {
		// Only accept events belonging to the same aggregate.
		if event.AggregateID() != aggregateID {
			return eh.OutboxError{
				Err:       eh.ErrInvalidEvent,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		// Only accept events that apply to the correct aggregate version.
		if event.Version() != version+1 {
			return eh.OutboxError{
				Err:       eh.ErrIncorrectEventVersion,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}

		// Create the event record for the DB.
		e, err := newEvt(ctx, event)
		if err != nil {
			return err
		}
		eventsToInsert[i] = event
		dbEvents[i] = *e
		version++
	}

	c := s.client.Database(s.dbName(ctx)).Collection("outbox")

	// Either insert a new aggregate or append to an existing.
	if originalVersion == 0 {
		aggregate := aggregateRecord{
			AggregateID: aggregateID,
			Version:     len(dbEvents),
			Events:      dbEvents,
		}

		if _, err := c.InsertOne(ctx, aggregate); err != nil {
			saveErr = eh.OutboxError{
				Err:       ErrCouldNotSaveAggregate,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
			return
		}
	} else {
		// Increment aggregate version on insert of new event record, and
		// only insert if version of aggregate is matching (ie not changed
		// since loading the aggregate).
		if r, err := c.UpdateOne(ctx,
			bson.M{
				"_id":     aggregateID,
				"version": originalVersion,
			},
			bson.M{
				"$push": bson.M{"events": bson.M{"$each": dbEvents}},
				"$inc":  bson.M{"version": len(dbEvents)},
			},
		); err != nil {
			saveErr = eh.OutboxError{
				Err:       ErrCouldNotSaveAggregate,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
			return
		} else if r.MatchedCount == 0 {
			saveErr = eh.OutboxError{
				Err:       ErrCouldNotSaveAggregate,
				BaseErr:   fmt.Errorf("incorrect inserted count"),
				Namespace: eh.NamespaceFromContext(ctx),
			}
			return
		}
	}

	// Let the optional event handler handle the events. Aborts the transaction
	// in case of error.
	if s.eventHandler != nil {
		for _, e := range eventsToInsert {
			if err := s.eventHandler.HandleEvent(ctx, e); err != nil {
				saveErr = eh.OutboxError{
					Err:       ErrCouldNotSaveAggregate,
					BaseErr:   err,
					Namespace: eh.NamespaceFromContext(ctx),
				}
				return
			}
		}
	}

	return nil
}

func (s *Outbox) Watch(ctx context.Context, h eh.EventHandler) ([]eh.Event, error) {
	// TODO: Read up all un-handled events and handle.
	// TODO: Watch outbox for changes.
	// TODO: Handle new events.
	// TODO: Mark events as handled.
	// TODO: Save resume token.

	c := s.client.Database(s.dbName(ctx)).Collection("outbox")

	var aggregate aggregateRecord
	err := c.FindOne(ctx, bson.M{"_id": id}).Decode(&aggregate)
	if err == mongo.ErrNoDocuments {
		return []eh.Event{}, nil
	} else if err != nil {
		return nil, eh.OutboxError{
			Err:       err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	events := make([]eh.Event, len(aggregate.Events))
	for i, dbEvent := range aggregate.Events {
		// Create an event of the correct type and decode from raw BSON.
		if len(dbEvent.RawData) > 0 {
			var err error
			if dbEvent.data, err = eh.CreateEventData(dbEvent.EventType); err != nil {
				return nil, eh.OutboxError{
					Err:       ErrCouldNotUnmarshalEvent,
					BaseErr:   err,
					Namespace: eh.NamespaceFromContext(ctx),
				}
			}
			if err := bson.Unmarshal(dbEvent.RawData, dbEvent.data); err != nil {
				return nil, eh.OutboxError{
					Err:       ErrCouldNotUnmarshalEvent,
					BaseErr:   err,
					Namespace: eh.NamespaceFromContext(ctx),
				}
			}
			dbEvent.RawData = nil
		}

		events[i] = event{dbEvent: dbEvent}
	}

	return events, nil
}

// Clear clears the event storage.
func (s *Outbox) Clear(ctx context.Context) error {
	c := s.client.Database(s.dbName(ctx)).Collection("events")

	if err := c.Drop(ctx); err != nil {
		return eh.OutboxError{
			Err:       ErrCouldNotClearDB,
			BaseErr:   err,
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}
	return nil
}

// Close closes the database client.
func (s *Outbox) Close(ctx context.Context) {
	s.client.Disconnect(ctx)
}

// aggregateRecord is the Database representation of an aggregate.
type aggregateRecord struct {
	AggregateID uuid.UUID `bson:"_id"`
	Version     int       `bson:"version"`
	Events      []evt     `bson:"events"`
	// Type        string        `bson:"type"`
	// Snapshot    bson.Raw      `bson:"snapshot"`
}

// evt is the internal event record for the MongoDB event store used
// to save and load events from the DB.
type evt struct {
	EventType     eh.EventType           `bson:"event_type"`
	RawData       bson.Raw               `bson:"data,omitempty"`
	data          eh.EventData           `bson:"-"`
	Timestamp     time.Time              `bson:"timestamp"`
	AggregateType eh.AggregateType       `bson:"aggregate_type"`
	AggregateID   uuid.UUID              `bson:"_id"`
	Version       int                    `bson:"version"`
	Metadata      map[string]interface{} `bson:"metadata"`
}

// newEvt returns a new evt for an event.
func newEvt(ctx context.Context, event eh.Event) (*evt, error) {
	e := &evt{
		EventType:     event.EventType(),
		Timestamp:     event.Timestamp(),
		AggregateType: event.AggregateType(),
		AggregateID:   event.AggregateID(),
		Version:       event.Version(),
		Metadata:      event.Metadata(),
	}

	// Marshal event data if there is any.
	if event.Data() != nil {
		var err error
		e.RawData, err = bson.Marshal(event.Data())
		if err != nil {
			return nil, eh.EventStoreError{
				Err:       ErrCouldNotMarshalEvent,
				BaseErr:   err,
				Namespace: eh.NamespaceFromContext(ctx),
			}
		}
	}

	return e, nil
}
