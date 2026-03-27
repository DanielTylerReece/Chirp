package testutil

import "time"

// EventSequence replays a series of events through a handler with configurable delays.
type EventSequence struct {
	events []scheduledEvent
}

type scheduledEvent struct {
	delay time.Duration
	event any
}

// NewEventSequence creates an empty EventSequence.
func NewEventSequence() *EventSequence {
	return &EventSequence{}
}

// Add appends an event to fire after the given delay.
func (es *EventSequence) Add(delay time.Duration, event any) *EventSequence {
	es.events = append(es.events, scheduledEvent{delay: delay, event: event})
	return es
}

// AddImmediate appends an event to fire with no delay.
func (es *EventSequence) AddImmediate(event any) *EventSequence {
	return es.Add(0, event)
}

// Play replays events through the given handler. Blocks until all events are dispatched.
func (es *EventSequence) Play(handler func(evt any)) {
	for _, se := range es.events {
		if se.delay > 0 {
			time.Sleep(se.delay)
		}
		handler(se.event)
	}
}

// PlayAsync replays events in a goroutine. Returns a channel that closes when done.
func (es *EventSequence) PlayAsync(handler func(evt any)) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		es.Play(handler)
		close(done)
	}()
	return done
}
