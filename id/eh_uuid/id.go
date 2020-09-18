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

package eh_uuid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"

	eh "github.com/looplab/eventhorizon"
)

// UseAsIDType sets this ID implementation to be used as default in Event Horizon.
func UseAsIDType() {
	eh.NewID = NewID
	eh.ParseID = ParseID
	eh.EmptyID = EmptyID
}

// Pattern used to parse hex string representation of the UUID.
// FIXME: do something to consider both brackets at one time,
// current one allows to parse string with only one opening
// or closing bracket.
const hexPattern = "^(urn\\:uuid\\:)?\\{?([a-f0-9]{8})-([a-f0-9]{4})-" +
	"([1-5][a-f0-9]{3})-([a-f0-9]{4})-([a-f0-9]{12})\\}?$"

var re = regexp.MustCompile(hexPattern)

// UUID is a unique identifier, based on the UUID spec. It must be exactly 16
// bytes long.
type id string

// EHID implements the EHID method of the eventhorizon.ID interface.
func (id) EHID() {}

// String implements the Stringer interface for UUID.
func (i id) String() string {
	return string(i)
}

// NewID creates a new ID with a UUID v4 string as the underlying type.
func NewID() eh.ID {
	var u [16]byte

	// Set all bits to randomly (or pseudo-randomly) chosen values.
	_, err := rand.Read(u[:])
	if err != nil {
		panic(err)
	}

	// Set the RFC4122 flag.
	u[8] = (u[8] & 0xBF) | 0x80

	// Set the version to 4.
	u[6] = (u[6] & 0xF) | 0x40

	return id(fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:]))
}

// ParseID creates a ID object from given hex string representation.
// The function accepts UUID string in following formats:
//
//     ParseUUID("6ba7b814-9dad-11d1-80b4-00c04fd430c8")
//     ParseUUID("{6ba7b814-9dad-11d1-80b4-00c04fd430c8}")
//     ParseUUID("urn:uuid:6ba7b814-9dad-11d1-80b4-00c04fd430c8")
//
func ParseID(s string) (eh.ID, error) {
	if s == "" {
		return EmptyID(), nil
	}

	md := re.FindStringSubmatch(s)
	if md == nil {
		return EmptyID(), errors.New("Invalid UUID string")
	}
	return id(fmt.Sprintf("%s-%s-%s-%s-%s", md[2], md[3], md[4], md[5], md[6])), nil
}

// EmptyID creates an ID with Google UUID as the underlying type.
func EmptyID() eh.ID {
	return id("")
}
