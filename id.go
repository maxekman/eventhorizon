package eventhorizon

func init() {
	NewID = missingNewID
	EmptyID = missingEmptyID
	ParseID = missingParseID
}

// ID is a ID of aggregates, entities etc.
type ID interface {
	// EHID is a marker to signal that a type can be used as an ID.
	EHID()

	// String returns a string representation of the ID.
	String() string

	UnmarshalJSON([]byte) error
}

// NewID creates a new ID.
var NewID func() ID

// EmptyID creates a empty ID.
var EmptyID func() ID

// ParseID creates a new ID from a string.
var ParseID func(string) (ID, error)

func missingNewID() ID {
	panic("eventhorizon: no ID implementation chosen")
}

func missingEmptyID() ID {
	// Don't panic here as it would not allow registering aggregates before
	// setting the ID type to use.
	return nil
}

func missingParseID(str string) (ID, error) {
	panic("eventhorizon: no ID implementation chosen")
}
