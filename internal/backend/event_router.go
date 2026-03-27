package backend

import (
	"github.com/tyler/gmessage/internal/app"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/events"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"
)

// EventRouter translates libgm events into EventBus events.
type EventRouter struct {
	bus *app.EventBus

	// Callback for session persistence when auth token is refreshed.
	OnAuthRefreshed func()
	// Callback when QR pairing succeeds.
	OnPairSuccess func()
	// Callback when a fatal listen error occurs (requires re-pair).
	OnFatalError func(error)
	// Callback when settings (including SIM info) are received.
	OnSettings func(*gmproto.Settings)
	// Callback when a message is received/updated (for persisting to DB).
	OnMessage func(msg *libgm.WrappedMessage)
}

// NewEventRouter creates an EventRouter bound to the given EventBus.
func NewEventRouter(bus *app.EventBus) *EventRouter {
	return &EventRouter{bus: bus}
}

// Handle processes a raw libgm event and publishes the appropriate typed event
// to the EventBus. Intended to be passed to RealClient.SetEventHandler.
func (er *EventRouter) Handle(evt any) {
	switch v := evt.(type) {
	case *events.ClientReady:
		er.bus.PublishStatus(app.StatusEvent{
			Connected:       true,
			PhoneResponding: true,
		})

	case *libgm.WrappedMessage:
		if er.OnMessage != nil {
			er.OnMessage(v)
		}
		er.bus.PublishMessage(app.MessageEvent{
			ConversationID: v.GetConversationID(),
			MessageID:      v.GetMessageID(),
			IsNew:          !v.IsOld,
		})

	case *gmproto.Conversation:
		er.bus.PublishConversation(app.ConversationEvent{
			ConversationID: v.GetConversationID(),
		})

	case *gmproto.TypingData:
		er.bus.PublishTyping(app.TypingEvent{
			ConversationID: v.GetConversationID(),
			IsTyping:       v.GetType() == gmproto.TypingTypes_STARTED_TYPING,
		})

	case *events.PairSuccessful:
		if er.OnPairSuccess != nil {
			er.OnPairSuccess()
		}

	case *events.AuthTokenRefreshed:
		if er.OnAuthRefreshed != nil {
			er.OnAuthRefreshed()
		}

	case *events.ListenFatalError:
		er.bus.PublishStatus(app.StatusEvent{
			Connected: false,
			Error:     v.Error.Error(),
		})
		if er.OnFatalError != nil {
			er.OnFatalError(v.Error)
		}

	case *events.ListenTemporaryError:
		er.bus.PublishStatus(app.StatusEvent{
			Connected: true,
			Error:     v.Error.Error(),
		})

	case *events.ListenRecovered:
		er.bus.PublishStatus(app.StatusEvent{
			Connected:       true,
			PhoneResponding: true,
		})

	case *events.PhoneNotResponding:
		er.bus.PublishStatus(app.StatusEvent{
			Connected:       true,
			PhoneResponding: false,
		})

	case *events.PhoneRespondingAgain:
		er.bus.PublishStatus(app.StatusEvent{
			Connected:       true,
			PhoneResponding: true,
		})

	case *gmproto.Settings:
		if er.OnSettings != nil {
			er.OnSettings(v)
		}
	}
}
