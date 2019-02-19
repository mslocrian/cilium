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

package eventqueue

import (
	"fmt"
	"testing"
	"time"

	"github.com/cilium/cilium/pkg/testutils"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type EventQueueSuite struct{}

var _ = Suite(&EventQueueSuite{})

func (s *EventQueueSuite) TestNewEventQueue(c *C) {
	q := NewEventQueue()
	c.Assert(q.close, Not(IsNil))
	c.Assert(q.events, Not(IsNil))
	c.Assert(q.drained, Not(IsNil))
	c.Assert(cap(q.events), Equals, 1)
}

func (s *EventQueueSuite) TestCloseEventQueueMultipleTimes(c *C) {
	q := NewEventQueue()
	q.CloseEventQueue()
	// Closing event queue twice should not cause panic.
	q.CloseEventQueue()
}

func (s *EventQueueSuite) TestNewEvent(c *C) {
	e := NewEvent(struct{}{})
	c.Assert(e.Metadata, Not(IsNil))
	c.Assert(e.EventResults, Not(IsNil))
	c.Assert(e.Cancelled, Not(IsNil))
}

type DummyEvent struct {}

func (d *DummyEvent) Handle() interface{} {
	return struct{}{}
}

type LongDummyEvent struct {}

func (l *LongDummyEvent) Handle() interface{} {
	time.Sleep(2*time.Second)
	return struct{}{}
}

func (s *EventQueueSuite) TestEventCancelAfterQueueClosed(c *C) {
	q := NewEventQueue()
	go q.RunEventQueue()
	ev := NewEvent(&DummyEvent{})
	q.QueueEvent(ev)

	// Event should not have been cancelled since queue was not closed.
	c.Assert(ev.WasCancelled(), Equals, false)
	q.CloseEventQueue()

	ev = NewEvent(&DummyEvent{})
	q.QueueEvent(ev)
	c.Assert(ev.WasCancelled(), Equals, true)
}

func (s *EventQueueSuite) TestBufferbloatScenario(c *C) {
	q := NewEventQueue()
	go q.RunEventQueue()
	ev := NewEvent(&LongDummyEvent{})
	ev2 := NewEvent(&LongDummyEvent{})
	q.QueueEvent(ev)
	go q.QueueEvent(ev2)
	q.CloseEventQueue()

	c.Assert(testutils.WaitUntil(func() bool {
		select {
		case <-q.drained:
			fmt.Printf("wait until queue drained\n")
			return true
		default:

			fmt.Printf("wait until queue NOT drained\n")
			return false
		}
	}, 10*time.Second), IsNil)

	// Event should be cancelled since it was in the queue.
	c.Assert(ev2.WasCancelled(), Equals, true)
	select {
	case <-ev2.EventResults:
		c.Error("event results channel should be close when cancelled")
	}



}
