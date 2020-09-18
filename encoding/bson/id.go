package mongodb

import (
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"

	eh "github.com/looplab/eventhorizon"
)

// Update the default BSON registry to be able to handle UUID types as strings.
func init() {
	rb := bson.NewRegistryBuilder()

	// Needs to be done like this to cover cases where the ID implementation
	// is based on a primitive type (e.g. `type id int64`).
	idType := reflect.TypeOf((*eh.ID)(nil)).Elem()
	rb.RegisterHookEncoder(idType, bsoncodec.ValueEncoderFunc(encodeID))
	rb.RegisterHookDecoder(idType, bsoncodec.ValueDecoderFunc(decodeID))

	bson.DefaultRegistry = rb.Build()
}

func encodeID(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	idType := reflect.TypeOf((*eh.ID)(nil)).Elem()
	if !val.IsValid() {
		// if !val.IsValid() || val.Kind() != idType.Kind() {
		return bsoncodec.ValueEncoderError{
			Name:     "eh.ID",
			Types:    []reflect.Type{idType},
			Received: val,
		}
	}
	id, ok := val.Interface().(eh.ID)
	if !ok {
		return bsoncodec.ValueEncoderError{
			Name:     "eh.ID",
			Types:    []reflect.Type{idType},
			Received: val,
		}
	}
	return vw.WriteString(id.String())
}

func decodeID(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	idType := reflect.TypeOf((*eh.ID)(nil)).Elem()
	if !val.IsValid() || !val.CanSet() || val.Kind() != idType.Kind() {
		return bsoncodec.ValueDecoderError{
			Name:     "eh.ID",
			Types:    []reflect.Type{idType},
			Received: val,
		}
	}

	var s string
	switch vr.Type() {
	case bsontype.String:
		var err error
		if s, err = vr.ReadString(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("received invalid BSON type to decode into UUID: %s", vr.Type())
	}

	id, err := eh.ParseID(s)
	if err != nil {
		return fmt.Errorf("could not parse UUID string: %s", s)
	}
	v := reflect.ValueOf(id)
	if !v.IsValid() {
		return fmt.Errorf("invalid reflected UUID value: %s", v.Kind().String())
	}
	val.Set(v)

	return nil
}
