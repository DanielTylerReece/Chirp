package app

// Event types

// MessageEvent signals that a message was received or updated.
type MessageEvent struct {
	ConversationID string
	MessageID      string
	IsNew          bool // false for backfill/old messages
}

// ConversationEvent signals that a conversation was created or updated.
type ConversationEvent struct {
	ConversationID string
}

// TypingEvent signals that a participant started or stopped typing.
type TypingEvent struct {
	ConversationID string
	ParticipantID  string
	IsTyping       bool
}

// StatusEvent signals a change in connection or phone status.
type StatusEvent struct {
	Connected       bool
	PhoneResponding bool
	Error           string
}

// EventSubscriber receives dispatched events from the EventBus.
type EventSubscriber interface {
	OnMessage(MessageEvent)
	OnConversation(ConversationEvent)
	OnTyping(TypingEvent)
	OnStatus(StatusEvent)
}

// EventBus fans out backend events to UI subscribers.
type EventBus struct {
	messages      chan MessageEvent
	conversations chan ConversationEvent
	typing        chan TypingEvent
	status        chan StatusEvent
	subscribers   []EventSubscriber
	done          chan struct{}

	// Lightweight callback subscribers (no interface needed)
	convCallbacks []func(ConversationEvent)
	msgCallbacks  []func(MessageEvent)
}

// NewEventBus creates an EventBus with buffered channels.
func NewEventBus() *EventBus {
	return &EventBus{
		messages:      make(chan MessageEvent, 64),
		conversations: make(chan ConversationEvent, 64),
		typing:        make(chan TypingEvent, 64),
		status:        make(chan StatusEvent, 16),
		done:          make(chan struct{}),
	}
}

// Subscribe registers a subscriber to receive all event types.
func (eb *EventBus) Subscribe(sub EventSubscriber) {
	eb.subscribers = append(eb.subscribers, sub)
}

// SubscribeConversation registers a lightweight callback for conversation events.
func (eb *EventBus) SubscribeConversation(fn func(ConversationEvent)) {
	eb.convCallbacks = append(eb.convCallbacks, fn)
}

// SubscribeMessage registers a lightweight callback for message events.
func (eb *EventBus) SubscribeMessage(fn func(MessageEvent)) {
	eb.msgCallbacks = append(eb.msgCallbacks, fn)
}

// Publish methods (called by backend)

// PublishMessage enqueues a message event. Drops if the channel is full.
func (eb *EventBus) PublishMessage(evt MessageEvent) {
	select {
	case eb.messages <- evt:
	default: // drop if full
	}
}

// PublishConversation enqueues a conversation event. Drops if the channel is full.
func (eb *EventBus) PublishConversation(evt ConversationEvent) {
	select {
	case eb.conversations <- evt:
	default:
	}
}

// PublishTyping enqueues a typing event. Drops if the channel is full.
func (eb *EventBus) PublishTyping(evt TypingEvent) {
	select {
	case eb.typing <- evt:
	default:
	}
}

// PublishStatus enqueues a status event. Drops if the channel is full.
func (eb *EventBus) PublishStatus(evt StatusEvent) {
	select {
	case eb.status <- evt:
	default:
	}
}

// Start begins dispatching events to subscribers. Call in a goroutine.
func (eb *EventBus) Start() {
	for {
		select {
		case evt := <-eb.messages:
			for _, sub := range eb.subscribers {
				sub.OnMessage(evt)
			}
			for _, fn := range eb.msgCallbacks {
				fn(evt)
			}
		case evt := <-eb.conversations:
			for _, sub := range eb.subscribers {
				sub.OnConversation(evt)
			}
			for _, fn := range eb.convCallbacks {
				fn(evt)
			}
		case evt := <-eb.typing:
			for _, sub := range eb.subscribers {
				sub.OnTyping(evt)
			}
		case evt := <-eb.status:
			for _, sub := range eb.subscribers {
				sub.OnStatus(evt)
			}
		case <-eb.done:
			return
		}
	}
}

// Stop shuts down the event bus.
func (eb *EventBus) Stop() {
	close(eb.done)
}
