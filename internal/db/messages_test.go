package db

import (
	"fmt"
	"testing"
)

func TestUpsertAndGetMessage(t *testing.T) {
	db := mustOpenTestDB(t)

	// Create conversation first (FK constraint)
	if err := db.UpsertConversation(&Conversation{ID: "conv-1", Name: "Test"}); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	msg := &Message{
		ID:             "msg-1",
		ConversationID: "conv-1",
		ParticipantID:  "part-1",
		Body:           "Hello, world!",
		TimestampMS:    1711400000000,
		IsFromMe:       true,
		Status:         2,
		Reactions:      `[{"emoji":"👍","from":"part-2"}]`,
	}

	if err := db.UpsertMessage(msg); err != nil {
		t.Fatalf("UpsertMessage: %v", err)
	}

	got, err := db.GetMessage("msg-1")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}

	if got.ID != msg.ID {
		t.Errorf("ID: got %q, want %q", got.ID, msg.ID)
	}
	if got.ConversationID != msg.ConversationID {
		t.Errorf("ConversationID: got %q, want %q", got.ConversationID, msg.ConversationID)
	}
	if got.ParticipantID != msg.ParticipantID {
		t.Errorf("ParticipantID: got %q, want %q", got.ParticipantID, msg.ParticipantID)
	}
	if got.Body != msg.Body {
		t.Errorf("Body: got %q, want %q", got.Body, msg.Body)
	}
	if got.TimestampMS != msg.TimestampMS {
		t.Errorf("TimestampMS: got %d, want %d", got.TimestampMS, msg.TimestampMS)
	}
	if got.IsFromMe != msg.IsFromMe {
		t.Errorf("IsFromMe: got %v, want %v", got.IsFromMe, msg.IsFromMe)
	}
	if got.Status != msg.Status {
		t.Errorf("Status: got %d, want %d", got.Status, msg.Status)
	}
	if got.Reactions != msg.Reactions {
		t.Errorf("Reactions: got %q, want %q", got.Reactions, msg.Reactions)
	}
}

func TestUpsertMessageWithMedia(t *testing.T) {
	db := mustOpenTestDB(t)

	if err := db.UpsertConversation(&Conversation{ID: "conv-1", Name: "Test"}); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	decryptKey := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	thumbKey := []byte{0xCA, 0xFE}

	msg := &Message{
		ID:              "msg-media",
		ConversationID:  "conv-1",
		Body:            "",
		TimestampMS:     1711400001000,
		IsFromMe:        false,
		Status:          2,
		MediaID:         "media-abc123",
		MediaMimeType:   "image/jpeg",
		MediaDecryptKey: decryptKey,
		MediaSize:       1048576,
		MediaWidth:      1920,
		MediaHeight:     1080,
		ThumbnailID:     "thumb-abc123",
		ThumbnailKey:    thumbKey,
		ReplyToID:       "msg-original",
		Reactions:       "[]",
	}

	if err := db.UpsertMessage(msg); err != nil {
		t.Fatalf("UpsertMessage: %v", err)
	}

	got, err := db.GetMessage("msg-media")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}

	if got.MediaID != "media-abc123" {
		t.Errorf("MediaID: got %q, want %q", got.MediaID, "media-abc123")
	}
	if got.MediaMimeType != "image/jpeg" {
		t.Errorf("MediaMimeType: got %q, want %q", got.MediaMimeType, "image/jpeg")
	}
	if len(got.MediaDecryptKey) != len(decryptKey) {
		t.Errorf("MediaDecryptKey length: got %d, want %d", len(got.MediaDecryptKey), len(decryptKey))
	}
	for i := range decryptKey {
		if got.MediaDecryptKey[i] != decryptKey[i] {
			t.Errorf("MediaDecryptKey[%d]: got %x, want %x", i, got.MediaDecryptKey[i], decryptKey[i])
			break
		}
	}
	if got.MediaSize != 1048576 {
		t.Errorf("MediaSize: got %d, want %d", got.MediaSize, 1048576)
	}
	if got.MediaWidth != 1920 {
		t.Errorf("MediaWidth: got %d, want %d", got.MediaWidth, 1920)
	}
	if got.MediaHeight != 1080 {
		t.Errorf("MediaHeight: got %d, want %d", got.MediaHeight, 1080)
	}
	if got.ThumbnailID != "thumb-abc123" {
		t.Errorf("ThumbnailID: got %q, want %q", got.ThumbnailID, "thumb-abc123")
	}
	if len(got.ThumbnailKey) != len(thumbKey) {
		t.Errorf("ThumbnailKey length: got %d, want %d", len(got.ThumbnailKey), len(thumbKey))
	}
	if got.ReplyToID != "msg-original" {
		t.Errorf("ReplyToID: got %q, want %q", got.ReplyToID, "msg-original")
	}
}

func TestGetMessagesPagination(t *testing.T) {
	db := mustOpenTestDB(t)

	if err := db.UpsertConversation(&Conversation{ID: "conv-1", Name: "Test"}); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	// Insert 10 messages with incrementing timestamps
	for i := 0; i < 10; i++ {
		msg := &Message{
			ID:             fmt.Sprintf("msg-%d", i),
			ConversationID: "conv-1",
			Body:           fmt.Sprintf("Message %d", i),
			TimestampMS:    int64(1000 + i*100),
			Reactions:      "[]",
		}
		if err := db.UpsertMessage(msg); err != nil {
			t.Fatalf("UpsertMessage %d: %v", i, err)
		}
	}

	// Get latest 5 (no timestamp filter)
	msgs, err := db.GetMessages("conv-1", 5, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
	// Should be ASC order (chronological): oldest of the latest 5 first
	if msgs[0].TimestampMS != 1500 {
		t.Errorf("first message timestamp: got %d, want 1500", msgs[0].TimestampMS)
	}
	if msgs[4].TimestampMS != 1900 {
		t.Errorf("last message timestamp: got %d, want 1900", msgs[4].TimestampMS)
	}

	// Get messages before timestamp 1500 (should get msgs with ts 1000-1400)
	msgs2, err := db.GetMessages("conv-1", 10, 1500)
	if err != nil {
		t.Fatalf("GetMessages with before: %v", err)
	}
	if len(msgs2) != 5 {
		t.Fatalf("expected 5 messages before ts 1500, got %d", len(msgs2))
	}
	if msgs2[0].TimestampMS != 1000 {
		t.Errorf("first message timestamp: got %d, want 1000", msgs2[0].TimestampMS)
	}
	if msgs2[4].TimestampMS != 1400 {
		t.Errorf("last message timestamp: got %d, want 1400", msgs2[4].TimestampMS)
	}
}

func TestUpdateMessageStatus(t *testing.T) {
	db := mustOpenTestDB(t)

	if err := db.UpsertConversation(&Conversation{ID: "conv-1", Name: "Test"}); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	msg := &Message{ID: "msg-1", ConversationID: "conv-1", Status: 0, Reactions: "[]"}
	if err := db.UpsertMessage(msg); err != nil {
		t.Fatalf("UpsertMessage: %v", err)
	}

	// Update to delivered
	if err := db.UpdateMessageStatus("msg-1", 2); err != nil {
		t.Fatalf("UpdateMessageStatus: %v", err)
	}

	got, err := db.GetMessage("msg-1")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got.Status != 2 {
		t.Errorf("Status: got %d, want 2", got.Status)
	}

	// Update to read
	if err := db.UpdateMessageStatus("msg-1", 3); err != nil {
		t.Fatalf("UpdateMessageStatus to read: %v", err)
	}
	got, err = db.GetMessage("msg-1")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got.Status != 3 {
		t.Errorf("Status: got %d, want 3", got.Status)
	}
}

func TestDeleteMessage(t *testing.T) {
	db := mustOpenTestDB(t)

	if err := db.UpsertConversation(&Conversation{ID: "conv-1", Name: "Test"}); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	msg := &Message{ID: "msg-1", ConversationID: "conv-1", Body: "delete me", Reactions: "[]"}
	if err := db.UpsertMessage(msg); err != nil {
		t.Fatalf("UpsertMessage: %v", err)
	}

	if err := db.DeleteMessage("msg-1"); err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}

	_, err := db.GetMessage("msg-1")
	if err == nil {
		t.Error("expected error getting deleted message")
	}
}

func TestDeleteMessageNotFound(t *testing.T) {
	db := mustOpenTestDB(t)

	err := db.DeleteMessage("nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent message")
	}
}

func TestCountMessages(t *testing.T) {
	db := mustOpenTestDB(t)

	if err := db.UpsertConversation(&Conversation{ID: "conv-1", Name: "Test"}); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	// Empty conversation
	count, err := db.CountMessages("conv-1")
	if err != nil {
		t.Fatalf("CountMessages: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// Add messages
	for i := 0; i < 3; i++ {
		msg := &Message{
			ID:             fmt.Sprintf("msg-%d", i),
			ConversationID: "conv-1",
			Body:           fmt.Sprintf("msg %d", i),
			Reactions:      "[]",
		}
		if err := db.UpsertMessage(msg); err != nil {
			t.Fatalf("UpsertMessage: %v", err)
		}
	}

	count, err = db.CountMessages("conv-1")
	if err != nil {
		t.Fatalf("CountMessages: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestGetMessageNotFound(t *testing.T) {
	db := mustOpenTestDB(t)

	_, err := db.GetMessage("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent message")
	}
}

func TestUpsertMessageUpdatesExisting(t *testing.T) {
	db := mustOpenTestDB(t)

	if err := db.UpsertConversation(&Conversation{ID: "conv-1", Name: "Test"}); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	msg := &Message{ID: "msg-1", ConversationID: "conv-1", Body: "original", Status: 0, Reactions: "[]"}
	if err := db.UpsertMessage(msg); err != nil {
		t.Fatalf("UpsertMessage (insert): %v", err)
	}

	msg.Body = "edited"
	msg.Status = 3
	if err := db.UpsertMessage(msg); err != nil {
		t.Fatalf("UpsertMessage (update): %v", err)
	}

	got, err := db.GetMessage("msg-1")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got.Body != "edited" {
		t.Errorf("Body: got %q, want %q", got.Body, "edited")
	}
	if got.Status != 3 {
		t.Errorf("Status: got %d, want 3", got.Status)
	}
}
