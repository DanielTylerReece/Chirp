package db

import (
	"database/sql"
	"fmt"
	"testing"
)

func mustOpenTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestUpsertAndGetConversation(t *testing.T) {
	db := mustOpenTestDB(t)

	conv := &Conversation{
		ID:                 "conv-1",
		Name:               "Alice",
		IsGroup:            false,
		LastMessageTS:      1711400005000,
		LastMessagePreview: "See you tomorrow!",
		UnreadCount:        1,
		IsPinned:           true,
		IsArchived:         false,
		IsRCS:              true,
		AvatarURL:          "https://example.com/alice.png",
		CreatedAt:          1711300000000,
	}

	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	got, err := db.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}

	if got.ID != conv.ID {
		t.Errorf("ID: got %q, want %q", got.ID, conv.ID)
	}
	if got.Name != conv.Name {
		t.Errorf("Name: got %q, want %q", got.Name, conv.Name)
	}
	if got.IsGroup != conv.IsGroup {
		t.Errorf("IsGroup: got %v, want %v", got.IsGroup, conv.IsGroup)
	}
	if got.LastMessageTS != conv.LastMessageTS {
		t.Errorf("LastMessageTS: got %d, want %d", got.LastMessageTS, conv.LastMessageTS)
	}
	if got.LastMessagePreview != conv.LastMessagePreview {
		t.Errorf("LastMessagePreview: got %q, want %q", got.LastMessagePreview, conv.LastMessagePreview)
	}
	if got.UnreadCount != conv.UnreadCount {
		t.Errorf("UnreadCount: got %d, want %d", got.UnreadCount, conv.UnreadCount)
	}
	if got.IsPinned != conv.IsPinned {
		t.Errorf("IsPinned: got %v, want %v", got.IsPinned, conv.IsPinned)
	}
	if got.IsArchived != conv.IsArchived {
		t.Errorf("IsArchived: got %v, want %v", got.IsArchived, conv.IsArchived)
	}
	if got.IsRCS != conv.IsRCS {
		t.Errorf("IsRCS: got %v, want %v", got.IsRCS, conv.IsRCS)
	}
	if got.AvatarURL != conv.AvatarURL {
		t.Errorf("AvatarURL: got %q, want %q", got.AvatarURL, conv.AvatarURL)
	}
	if got.CreatedAt != conv.CreatedAt {
		t.Errorf("CreatedAt: got %d, want %d", got.CreatedAt, conv.CreatedAt)
	}
}

func TestUpsertConversationUpdatesExisting(t *testing.T) {
	db := mustOpenTestDB(t)

	conv := &Conversation{ID: "conv-1", Name: "Alice", CreatedAt: 1000}
	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation (insert): %v", err)
	}

	conv.Name = "Alice Updated"
	conv.UnreadCount = 5
	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation (update): %v", err)
	}

	got, err := db.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.Name != "Alice Updated" {
		t.Errorf("Name: got %q, want %q", got.Name, "Alice Updated")
	}
	if got.UnreadCount != 5 {
		t.Errorf("UnreadCount: got %d, want %d", got.UnreadCount, 5)
	}
}

func TestListConversationsSorting(t *testing.T) {
	db := mustOpenTestDB(t)

	// Insert: unpinned recent, pinned old, unpinned old
	convs := []*Conversation{
		{ID: "conv-recent", Name: "Recent Unpinned", LastMessageTS: 3000, IsPinned: false},
		{ID: "conv-pinned", Name: "Pinned Old", LastMessageTS: 1000, IsPinned: true},
		{ID: "conv-old", Name: "Old Unpinned", LastMessageTS: 2000, IsPinned: false},
	}
	for _, c := range convs {
		if err := db.UpsertConversation(c); err != nil {
			t.Fatalf("UpsertConversation %s: %v", c.ID, err)
		}
	}

	got, err := db.ListConversations(10, 0)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 conversations, got %d", len(got))
	}

	// Pinned should be first regardless of timestamp
	if got[0].ID != "conv-pinned" {
		t.Errorf("first should be pinned, got %q", got[0].ID)
	}
	// Then by timestamp DESC
	if got[1].ID != "conv-recent" {
		t.Errorf("second should be recent unpinned, got %q", got[1].ID)
	}
	if got[2].ID != "conv-old" {
		t.Errorf("third should be old unpinned, got %q", got[2].ID)
	}
}

func TestListConversationsPagination(t *testing.T) {
	db := mustOpenTestDB(t)

	for i := 0; i < 5; i++ {
		c := &Conversation{ID: fmt.Sprintf("conv-%d", i), LastMessageTS: int64(i * 1000)}
		if err := db.UpsertConversation(c); err != nil {
			t.Fatalf("UpsertConversation: %v", err)
		}
	}

	got, err := db.ListConversations(2, 0)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}

	got2, err := db.ListConversations(2, 2)
	if err != nil {
		t.Fatalf("ListConversations offset: %v", err)
	}
	if len(got2) != 2 {
		t.Errorf("expected 2, got %d", len(got2))
	}
	// Ensure different results
	if got[0].ID == got2[0].ID {
		t.Error("pagination returned same first result")
	}
}

func TestUpsertAndGetParticipants(t *testing.T) {
	db := mustOpenTestDB(t)

	conv := &Conversation{ID: "conv-1", Name: "Test Conv"}
	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	p1 := &Participant{
		ID:             "part-me",
		ConversationID: "conv-1",
		Name:           "Tyler",
		PhoneNumber:    "+15551234567",
		IsMe:           true,
		AvatarHexColor: "#FF5733",
	}
	// Create the contact so the FK on contact_id is satisfied
	if err := db.UpsertContact(&Contact{ID: "contact-1", Name: "Alice", PhoneNumber: "+15559876543"}); err != nil {
		t.Fatalf("UpsertContact: %v", err)
	}

	p2 := &Participant{
		ID:             "part-alice",
		ConversationID: "conv-1",
		ContactID:      sql.NullString{String: "contact-1", Valid: true},
		Name:           "Alice",
		PhoneNumber:    "+15559876543",
		IsMe:           false,
		AvatarHexColor: "#33FF57",
	}

	if err := db.UpsertParticipant(p1); err != nil {
		t.Fatalf("UpsertParticipant p1: %v", err)
	}
	if err := db.UpsertParticipant(p2); err != nil {
		t.Fatalf("UpsertParticipant p2: %v", err)
	}

	parts, err := db.GetParticipants("conv-1")
	if err != nil {
		t.Fatalf("GetParticipants: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(parts))
	}

	// Find the "me" participant
	var me *Participant
	for i := range parts {
		if parts[i].IsMe {
			me = &parts[i]
			break
		}
	}
	if me == nil {
		t.Fatal("no participant with IsMe=true found")
	}
	if me.Name != "Tyler" {
		t.Errorf("me.Name: got %q, want %q", me.Name, "Tyler")
	}
	if me.AvatarHexColor != "#FF5733" {
		t.Errorf("me.AvatarHexColor: got %q, want %q", me.AvatarHexColor, "#FF5733")
	}
}

func TestGetConversationIncludesParticipants(t *testing.T) {
	db := mustOpenTestDB(t)

	conv := &Conversation{ID: "conv-1", Name: "Test"}
	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}
	if err := db.UpsertParticipant(&Participant{ID: "p1", ConversationID: "conv-1", Name: "Alice"}); err != nil {
		t.Fatalf("UpsertParticipant: %v", err)
	}

	got, err := db.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if len(got.Participants) != 1 {
		t.Errorf("expected 1 participant, got %d", len(got.Participants))
	}
}

func TestDeleteConversationCascades(t *testing.T) {
	db := mustOpenTestDB(t)

	conv := &Conversation{ID: "conv-1", Name: "Test"}
	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	// Add participant
	if err := db.UpsertParticipant(&Participant{ID: "p1", ConversationID: "conv-1", Name: "Alice"}); err != nil {
		t.Fatalf("UpsertParticipant: %v", err)
	}

	// Add message
	if err := db.UpsertMessage(&Message{ID: "msg-1", ConversationID: "conv-1", Body: "Hello"}); err != nil {
		t.Fatalf("UpsertMessage: %v", err)
	}

	// Delete conversation
	if err := db.DeleteConversation("conv-1"); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}

	// Verify conversation gone
	_, err := db.GetConversation("conv-1")
	if err == nil {
		t.Error("expected error getting deleted conversation")
	}

	// Verify participants gone
	parts, err := db.GetParticipants("conv-1")
	if err != nil {
		t.Fatalf("GetParticipants after delete: %v", err)
	}
	if len(parts) != 0 {
		t.Errorf("expected 0 participants after cascade delete, got %d", len(parts))
	}

	// Verify messages gone
	count, err := db.CountMessages("conv-1")
	if err != nil {
		t.Fatalf("CountMessages after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 messages after cascade delete, got %d", count)
	}
}

func TestUpdateConversationLastMessage(t *testing.T) {
	db := mustOpenTestDB(t)

	conv := &Conversation{ID: "conv-1", Name: "Test", LastMessageTS: 1000, LastMessagePreview: "old"}
	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	if err := db.UpdateConversationLastMessage("conv-1", 2000, "new preview"); err != nil {
		t.Fatalf("UpdateConversationLastMessage: %v", err)
	}

	got, err := db.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.LastMessageTS != 2000 {
		t.Errorf("LastMessageTS: got %d, want 2000", got.LastMessageTS)
	}
	if got.LastMessagePreview != "new preview" {
		t.Errorf("LastMessagePreview: got %q, want %q", got.LastMessagePreview, "new preview")
	}
}

func TestUpdateConversationUnread(t *testing.T) {
	db := mustOpenTestDB(t)

	conv := &Conversation{ID: "conv-1", Name: "Test"}
	if err := db.UpsertConversation(conv); err != nil {
		t.Fatalf("UpsertConversation: %v", err)
	}

	if err := db.UpdateConversationUnread("conv-1", 7); err != nil {
		t.Fatalf("UpdateConversationUnread: %v", err)
	}

	got, err := db.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.UnreadCount != 7 {
		t.Errorf("UnreadCount: got %d, want 7", got.UnreadCount)
	}
}

func TestGetConversationNotFound(t *testing.T) {
	db := mustOpenTestDB(t)

	_, err := db.GetConversation("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent conversation")
	}
}

func TestDeleteConversationNotFound(t *testing.T) {
	db := mustOpenTestDB(t)

	err := db.DeleteConversation("nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent conversation")
	}
}
