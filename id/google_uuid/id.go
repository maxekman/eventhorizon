package google_uuid

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	eh "github.com/looplab/eventhorizon"
)

// UseAsIDType sets this ID implementation to be used as default in Event Horizon.
func UseAsIDType() {
	eh.NewID = func() eh.ID {
		return NewID()
	}
	eh.EmptyID = func() eh.ID {
		return EmptyID()
	}
	eh.ParseID = func(s string) (eh.ID, error) {
		return ParseID(s)
	}
}

// ID is a github.com/google/uuid implementation of eventhorizon.ID.
type ID uuid.UUID

// EHID implements the EHID method of the eventhorizon.ID interface.
func (ID) EHID() {}

// String implements the String method of the eventhorizon.ID interface.
func (i ID) String() string { return uuid.UUID(i).String() }

// NewID creates a new ID with Google UUID as the underlying type.
func NewID() ID {
	return ID(uuid.New())
}

// EmptyID creates an ID with Google UUID as the underlying type.
func EmptyID() ID {
	return ID(uuid.Nil)
}

// ParseID parses a ID from a string with Google UUID as the underlying type.
func ParseID(str string) (ID, error) {
	if str == "" {
		return EmptyID(), nil
	}
	i, err := uuid.Parse(str)
	if err != nil {
		return EmptyID(), errors.New("Invalid UUID string")
	}
	return ID(i), nil
}

// MarshalJSON turns UUID into a json.Marshaller.
func (i ID) MarshalJSON() ([]byte, error) {
	if i == EmptyID() {
		return []byte(`""`), nil
	}
	// Pack the string representation in quotes
	return []byte(fmt.Sprintf(`"%s"`, i.String())), nil
}

// UnmarshalJSON turns *UUID into a json.Unmarshaller.
func (i *ID) UnmarshalJSON(data []byte) error {
	// Data is expected to be a json string, like: "819c4ff4-31b4-4519-5d24-3c4a129b8649"
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid UUID in JSON, %v is not a valid JSON string", string(data))
	}

	// Grab string value without the surrounding " characters
	value := string(data[1 : len(data)-1])
	parsed, err := ParseID(value)
	if err != nil {
		return fmt.Errorf("invalid UUID in JSON, %v: %v", value, err)
	}

	// Dereference pointer value and store parsed
	*i = parsed
	return nil
}
