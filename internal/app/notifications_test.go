package app

import (
	"testing"

	"github.com/tyler/gmessage/internal/db"
)

func TestShouldNotify_SkipsOwnMessages(t *testing.T) {
	nm := &NotificationManager{enabled: true}

	msg := &db.Message{
		ID:             "msg-1",
		ConversationID: "conv-1",
		Body:           "Hello",
		IsFromMe:       true,
	}

	if nm.shouldNotify(msg) {
		t.Error("should not notify for own messages")
	}
}

func TestShouldNotify_SkipsActiveConversation(t *testing.T) {
	nm := &NotificationManager{
		enabled:      true,
		activeConvID: "conv-1",
		appFocused:   true,
	}

	msg := &db.Message{
		ID:             "msg-1",
		ConversationID: "conv-1",
		Body:           "Hello",
		IsFromMe:       false,
	}

	if nm.shouldNotify(msg) {
		t.Error("should not notify for active focused conversation")
	}
}

func TestShouldNotify_NotifiesWhenUnfocused(t *testing.T) {
	nm := &NotificationManager{
		enabled:      true,
		activeConvID: "conv-1",
		appFocused:   false,
	}

	msg := &db.Message{
		ID:             "msg-1",
		ConversationID: "conv-1",
		Body:           "Hello",
		IsFromMe:       false,
	}

	if !nm.shouldNotify(msg) {
		t.Error("should notify when app is not focused")
	}
}

func TestShouldNotify_NotifiesForDifferentConversation(t *testing.T) {
	nm := &NotificationManager{
		enabled:      true,
		activeConvID: "conv-1",
		appFocused:   true,
	}

	msg := &db.Message{
		ID:             "msg-2",
		ConversationID: "conv-2",
		Body:           "Hey",
		IsFromMe:       false,
	}

	if !nm.shouldNotify(msg) {
		t.Error("should notify for a different conversation even when focused")
	}
}

func TestShouldNotify_Disabled(t *testing.T) {
	nm := &NotificationManager{enabled: false}

	msg := &db.Message{
		IsFromMe: false,
	}

	if nm.shouldNotify(msg) {
		t.Error("should not notify when disabled")
	}
}

func TestTruncateNotification_Short(t *testing.T) {
	got := truncateNotification("short", 100)
	if got != "short" {
		t.Errorf("expected %q, got %q", "short", got)
	}
}

func TestTruncateNotification_ExactLength(t *testing.T) {
	s := "exactly10!"
	got := truncateNotification(s, 10)
	if got != s {
		t.Errorf("expected %q, got %q", s, got)
	}
}

func TestTruncateNotification_Long(t *testing.T) {
	long := "This is a really long message that should be truncated to fit within the notification body limit"
	got := truncateNotification(long, 50)
	runes := []rune(got)
	if len(runes) > 50 {
		t.Errorf("expected max 50 runes, got %d", len(runes))
	}
	if got[len(got)-3:] != "..." {
		t.Errorf("expected trailing '...', got %q", got[len(got)-3:])
	}
}

func TestTruncateNotification_Empty(t *testing.T) {
	got := truncateNotification("", 100)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestTruncateNotification_MultiByte(t *testing.T) {
	// Ensure we don't split multi-byte runes.
	s := "Hello, !!!!" // 11 runes
	got := truncateNotification(s, 10)
	runes := []rune(got)
	if len(runes) > 10 {
		t.Errorf("expected max 10 runes, got %d", len(runes))
	}
}
