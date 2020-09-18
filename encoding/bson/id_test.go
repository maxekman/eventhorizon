package mongodb

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/id/eh_uuid"
	"github.com/looplab/eventhorizon/id/google_uuid"
	"github.com/looplab/eventhorizon/mocks"
)

func TestGoogleUUID(t *testing.T) {
	google_uuid.UseAsIDType()

	model := &mocks.Model{
		ID: eh.NewID(),
	}
	b, err := bson.Marshal(model)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	decodedModel := &mocks.Model{}
	if err := bson.Unmarshal(b, &decodedModel); err != nil {
		t.Error("there should be no error:", err)
	}

	if model.ID != decodedModel.ID {
		t.Error("the ID should be marshaled and unmarshaled correctly")
	}
}

func TestEventHorizonUUID(t *testing.T) {
	eh_uuid.UseAsIDType()

	model := &mocks.Model{
		ID: eh.NewID(),
	}
	b, err := bson.Marshal(model)
	if err != nil {
		t.Error("there should be no error:", err)
	}

	decodedModel := &mocks.Model{}
	if err := bson.Unmarshal(b, &decodedModel); err != nil {
		t.Error("there should be no error:", err)
	}

	if model.ID != decodedModel.ID {
		t.Error("the ID should be marshaled and unmarshaled correctly")
	}
}
