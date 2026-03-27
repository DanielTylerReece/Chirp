package backend

import (
	"fmt"
	"log"
	"time"

	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/db"
)

// MessageHandler processes incoming message events and persists them.
type MessageHandler struct {
	database *db.DB
	bus      *app.EventBus
}

func NewMessageHandler(database *db.DB, bus *app.EventBus) *MessageHandler {
	return &MessageHandler{
		database: database,
		bus:      bus,
	}
}

// HandleNewMessage processes a new message received from libgm.
// Called by the EventRouter when a WrappedMessage arrives.
func (mh *MessageHandler) HandleNewMessage(msg *db.Message, convID string, isNew bool) {
	// Upsert the message
	if err := mh.database.UpsertMessage(msg); err != nil {
		log.Printf("message_handler: upsert message %s: %v", msg.ID, err)
		return
	}

	// Update conversation's last message
	if err := mh.database.UpdateConversationLastMessage(convID, msg.TimestampMS, truncate(msg.Body, 100)); err != nil {
		log.Printf("message_handler: update conv last message: %v", err)
	}

	// Index for FTS
	if msg.Body != "" {
		if err := mh.database.IndexMessage(msg.ID, msg.Body); err != nil {
			log.Printf("message_handler: index message: %v", err)
		}
	}

	// Emit event for UI
	mh.bus.PublishMessage(app.MessageEvent{
		ConversationID: convID,
		MessageID:      msg.ID,
		IsNew:          isNew,
	})
}

// HandleConversationUpdate processes a conversation update from libgm.
func (mh *MessageHandler) HandleConversationUpdate(conv *db.Conversation) {
	if err := mh.database.UpsertConversation(conv); err != nil {
		log.Printf("message_handler: upsert conversation %s: %v", conv.ID, err)
		return
	}

	mh.bus.PublishConversation(app.ConversationEvent{
		ConversationID: conv.ID,
	})
}

// HandleContactUpdate processes contact list update.
func (mh *MessageHandler) HandleContactUpdate(contact *db.Contact) {
	if err := mh.database.UpsertContact(contact); err != nil {
		log.Printf("message_handler: upsert contact %s: %v", contact.ID, err)
		return
	}
}

// SendOptimistic inserts a message with status=sending before the server confirms.
// Returns the temporary message ID.
func (mh *MessageHandler) SendOptimistic(convID string, body string) string {
	tmpID := fmt.Sprintf("tmp_%d", time.Now().UnixNano())
	msg := &db.Message{
		ID:             tmpID,
		ConversationID: convID,
		Body:           body,
		TimestampMS:    time.Now().UnixMilli(),
		IsFromMe:       true,
		Status:         0, // sending
	}

	if err := mh.database.UpsertMessage(msg); err != nil {
		log.Printf("message_handler: optimistic insert: %v", err)
		return ""
	}

	mh.bus.PublishMessage(app.MessageEvent{
		ConversationID: convID,
		MessageID:      tmpID,
		IsNew:          true,
	})

	return tmpID
}

// ConfirmSent updates a tmp_ message to its real ID and status.
func (mh *MessageHandler) ConfirmSent(tmpID, realID string, status int) {
	// Delete the tmp message
	if err := mh.database.DeleteMessage(tmpID); err != nil {
		log.Printf("message_handler: delete tmp message %s: %v", tmpID, err)
	}

	// The real message will come through as a WrappedMessage event
	// and be upserted normally
}

// MarkFailed marks a message as failed to send.
func (mh *MessageHandler) MarkFailed(msgID string) {
	if err := mh.database.UpdateMessageStatus(msgID, 4); err != nil {
		log.Printf("message_handler: mark failed: %v", err)
	}
	// Re-publish so UI updates
	msg, err := mh.database.GetMessage(msgID)
	if err == nil && msg != nil {
		mh.bus.PublishMessage(app.MessageEvent{
			ConversationID: msg.ConversationID,
			MessageID:      msgID,
			IsNew:          false,
		})
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
