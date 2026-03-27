package app

import (
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/tyler/gmessage/internal/db"
)

// NotificationManager handles desktop notifications for incoming messages.
type NotificationManager struct {
	app          *gio.Application
	database     *db.DB
	enabled      bool
	activeConvID string // don't notify for the conversation currently viewed
	appFocused   bool
}

// NewNotificationManager creates a NotificationManager bound to a GApplication.
func NewNotificationManager(app *gio.Application, database *db.DB) *NotificationManager {
	return &NotificationManager{
		app:      app,
		database: database,
		enabled:  true,
	}
}

// SetEnabled enables or disables notifications.
func (nm *NotificationManager) SetEnabled(enabled bool) {
	nm.enabled = enabled
}

// SetActiveConversation sets which conversation is currently viewed.
func (nm *NotificationManager) SetActiveConversation(convID string) {
	nm.activeConvID = convID
}

// SetAppFocused sets whether the app window is focused.
func (nm *NotificationManager) SetAppFocused(focused bool) {
	nm.appFocused = focused
}

// shouldNotify returns true if a notification should be sent for this message.
// Extracted from NotifyNewMessage so it can be tested without a GTK runtime.
func (nm *NotificationManager) shouldNotify(msg *db.Message) bool {
	if !nm.enabled {
		return false
	}
	if msg.IsFromMe {
		return false
	}
	if nm.appFocused && nm.activeConvID == msg.ConversationID {
		return false
	}
	return true
}

// NotifyNewMessage sends a desktop notification for a new incoming message.
// Returns true if a notification was sent.
func (nm *NotificationManager) NotifyNewMessage(msg *db.Message, conv *db.Conversation) bool {
	if !nm.shouldNotify(msg) {
		return false
	}

	title := conv.Name
	body := truncateNotification(msg.Body, 100)
	if body == "" && msg.MediaID != "" {
		body = "Sent a photo"
	}

	notification := gio.NewNotification(title)
	notification.SetBody(body)

	// Use conversation ID as notification ID so subsequent messages from the
	// same conversation replace the previous notification instead of stacking.
	nm.app.SendNotification(msg.ConversationID, notification)

	return true
}

// Withdraw removes a notification, e.g. when the user opens the conversation.
func (nm *NotificationManager) Withdraw(convID string) {
	nm.app.WithdrawNotification(convID)
}

// truncateNotification shortens s to maxLen, appending "..." if truncated.
// Operates on runes to avoid splitting multi-byte UTF-8 characters.
func truncateNotification(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
