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

package google_uuid

import (
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewID(t *testing.T) {
	id1 := NewID()
	if id1 == EmptyID() {
		t.Error("there should be a ID")
	}
	id2 := NewID()
	if id1 == id2 {
		t.Error("the IDs should be unique")
	}

	parts := strings.Split(id1.String(), "-")
	hash := parts[0] + parts[1] + parts[2] + parts[3] + parts[4]
	b, err := hex.DecodeString(hash)
	if err != nil {
		t.Error(err)
	}

	if (b[8]&0xC0)|0x80 != uint8(0x80) {
		t.Error("the variant should be correct")
	}

	if b[6]>>4 != uint8(4) {
		t.Error("the version should be correct")
	}

	re := regexp.MustCompile("^[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$")
	if !re.MatchString(id1.String()) {
		t.Error("the string format should be correct:", id1)
	}
}

func TestParseID(t *testing.T) {
	id := ID(uuid.MustParse("a4da289d-466d-4a56-4521-1dbd455aa0cd"))

	parsed, err := ParseID("")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if parsed != EmptyID() {
		t.Error("the ID should be correct:", parsed)
	}

	parsed, err = ParseID("a4da289d-466d-4a56-4521-1dbd455aa0cd")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if parsed != id {
		t.Error("the ID should be correct:", parsed)
	}

	parsed, err = ParseID("{a4da289d-466d-4a56-4521-1dbd455aa0cd}")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if parsed != id {
		t.Error("the ID should be correct:", parsed)
	}

	parsed, err = ParseID("urn:uuid:a4da289d-466d-4a56-4521-1dbd455aa0cd")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if parsed != id {
		t.Error("the ID should be correct:", parsed)
	}

	parsed, err = ParseID("not-a-uuid")
	if err == nil || err.Error() != "Invalid UUID string" {
		t.Error("there should be a 'Invalid UUID string' error:", err)
	}
	if parsed != EmptyID() {
		t.Error("the ID should be empty:", parsed)
	}
}

func TestString(t *testing.T) {
	id := ID(uuid.MustParse("a4da289d-466d-4a56-4521-1dbd455aa0cd"))
	if id.String() != "a4da289d-466d-4a56-4521-1dbd455aa0cd" {
		t.Error("the ID should be correct:", id)
	}
}

type jsonType struct {
	ID ID
}

func TestMarshalJSON(t *testing.T) {
	// Empty UUID.
	id, err := ParseID("")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	v := jsonType{ID: id}
	js, err := json.Marshal(&v)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if string(js) != `{"ID":""}` {
		t.Error("the JSON should be correct:", string(js))
	}

	// Normal UUID.
	id, err = ParseID("a4da289d-466d-4a56-4521-1dbd455aa0cd")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	v = jsonType{ID: id}
	js, err = json.Marshal(&v)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if string(js) != `{"ID":"a4da289d-466d-4a56-4521-1dbd455aa0cd"}` {
		t.Error("the JSON should be correct:", string(js))
	}
}

func TestUnmarshalJSON(t *testing.T) {
	// Empty UUID.
	js := []byte(`{"ID":""}`)
	v := jsonType{}
	err := json.Unmarshal(js, &v)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	id, err := ParseID("")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if v.ID != id {
		t.Error("the ID should be correct:", v.ID)
	}

	// Normal UUID.
	js = []byte(`{"ID":"a4da289d-466d-4a56-4521-1dbd455aa0cd"}`)
	v = jsonType{}
	err = json.Unmarshal(js, &v)
	if err != nil {
		t.Error("there should be no error:", err)
	}
	id, err = ParseID("a4da289d-466d-4a56-4521-1dbd455aa0cd")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if v.ID != id {
		t.Error("the ID should be correct:", v.ID)
	}
}

func TestUnmarshalJSONError(t *testing.T) {
	v := jsonType{}
	err := json.Unmarshal([]byte(`{"ID":"not-uuid"}`), &v)
	if err == nil || err.Error() != "invalid UUID in JSON, not-uuid: Invalid UUID string" {
		t.Error("there should be a 'invalid UUID in JSON, not-uuid: Invalid UUID string' error:", err)
	}

	err = json.Unmarshal([]byte(`{"ID":"819c4ff4-31b4-4519-xxxx-3c4a129b8649"}`), &v)
	if err == nil || err.Error() != "invalid UUID in JSON, 819c4ff4-31b4-4519-xxxx-3c4a129b8649: Invalid UUID string" {
		t.Error("there should be a 'invalid UUID in JSON, 819c4ff4-31b4-4519-xxxx-3c4a129b8649: Invalid UUID string' error:", err)
	}
}
