// Copyright 2019 Authors of Cilium
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

package endpoint

// EndpointEvent is an event that can be queued for an Endpoint on its
// EventQueue.
type EndpointEvent struct {
	// EndpointEventMetadata is the information about the event which is sent
	// by its queuer.
	EndpointEventMetadata interface{}

	// EventResults is a channel on which the results of the event are sent.
	// It is populated by the EventQueue itself, not by the queuer.
	EventResults chan interface{}

	// Cancelled is a channel which is called when the EventQueue is being drained.
	// The event was not ran if it was signaled upon.
	Cancelled chan struct{}
}

// NewEndpointEvent returns an EndpointEvent with all fields initialized.
func NewEndpointEvent(meta interface{}) *EndpointEvent {
	return &EndpointEvent{
		EndpointEventMetadata: meta,
		EventResults:          make(chan interface{}, 1),
		Cancelled:             make(chan struct{}),
	}
}

// EndpointRegenerationEvent contains all fields necessary to regenerate an endpoint.
type EndpointRegenerationEvent struct {
	owner        Owner
	regenContext *regenerationContext
	ep           *Endpoint
}

func (ev *EndpointRegenerationEvent) Handle() interface{} {
	err := ev.ep.regenerate(ev.owner, ev.regenContext)
	return &EndpointRegenerationResult{
		err: err,
	}
}

// EndpointRegenerationResult contains the results of an endpoint regeneration.
type EndpointRegenerationResult struct {
	err error
}

// EndpointRevisionBumpEvent contains all fields necessary to bump the policy
// revision of a given endpoint.
type EndpointRevisionBumpEvent struct {
	Rev uint64
	ep  *Endpoint
}

func (ev *EndpointRevisionBumpEvent) Handle() interface{} {
	// TODO: if the endpoint is not in a 'ready' state that means that
	// we cannot set the policy revision, as something else has
	// changed endpoint state which necessitates regeneration,
	// *or* the endpoint is in a not-ready state (i.e., a prior
	// regeneration failed, so there is no way that we can
	// realize the policy revision yet. Should this be signaled
	// to the routine waiting for the result of this event?
	ev.ep.getLogger().Debug("received endpoint revision bump event")
	ev.ep.SetPolicyRevision(ev.Rev)
	ev.ep.getLogger().Debug("sending endpoint revision bump result")
	return struct{}{}
}
