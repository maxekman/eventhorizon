package google_uuid

import (
	"fmt"

	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
)

// UseAsIDType sets this ID implementation to be used as default in Event Horizon.
func UseAsIDType() {
	eh.NewID = NewID
	eh.ParseID = ParseID
	eh.EmptyID = EmptyID
}

// id is a github.com/google/uuid implementation of eventhorizon.ID.
type id uuid.UUID

// EHID implements the EHID method of the eventhorizon.ID interface.
func (id) EHID() {}

// String implements the String method of the eventhorizon.ID interface.
func (i id) String() string { return uuid.UUID(i).String() }

// NewID creates a new ID with Google UUID as the underlying type.
func NewID() eh.ID {
	return id(uuid.New())
}

// ParseID parses a ID from a string with Google UUID as the underlying type.
func ParseID(str string) (eh.ID, error) {
	i, err := uuid.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("could not parse ID string: %w", err)
	}
	return id(i), nil
}

// EmptyID creates an ID with Google UUID as the underlying type.
func EmptyID() eh.ID {
	return id(uuid.Nil)
}
