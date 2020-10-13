// Copyright (c) 2020 - The Event Horizon authors.
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

package eventhorizon

// Outbox is an outbox for events, used to ensure handling/publishing of stored
// events. Should typically be used as a event handler for an event store which
// should be run inside (and use) a transaction.
type Outbox interface {
	EventHandler

	// Watch watches the outbox for new events and lets the event handler handle
	// them. Should only mark an event as handled if the handler doesn't return
	// an error. As a consequence it should guarantee at least once delivery to
	// event handler, with the possibility to resume the watch.
	Watch(EventHandler) error
}
