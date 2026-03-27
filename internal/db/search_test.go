package db

import (
	"fmt"
	"testing"
)

// insertTestConversation creates a conversation for FK constraints in tests.
func insertTestConversation(t *testing.T, db *DB, id string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO conversations (id, name) VALUES (?, ?)`, id, "Test Conv")
	if err != nil {
		t.Fatalf("insert conversation %s: %v", id, err)
	}
}

// insertTestParticipant creates a participant for join tests.
func insertTestParticipant(t *testing.T, db *DB, id, convID, name string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO participants (id, conversation_id, name) VALUES (?, ?, ?)`, id, convID, name)
	if err != nil {
		t.Fatalf("insert participant %s: %v", id, err)
	}
}

// insertTestMessage inserts a message into the messages table.
func insertTestMessage(t *testing.T, db *DB, id, convID, participantID, body string, tsMS int64, isFromMe int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO messages (id, conversation_id, participant_id, body, timestamp_ms, is_from_me)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, convID, participantID, body, tsMS, isFromMe)
	if err != nil {
		t.Fatalf("insert message %s: %v", id, err)
	}
}

func TestSearchRebuildFTS(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	insertTestConversation(t, db, "conv-1")
	insertTestParticipant(t, db, "part-1", "conv-1", "Alice")

	// Insert 10 messages
	for i := 0; i < 10; i++ {
		body := fmt.Sprintf("message number %d about networking", i)
		if i == 5 {
			body = "this one is about firewalls and security"
		}
		insertTestMessage(t, db, fmt.Sprintf("msg-%d", i), "conv-1", "part-1", body, int64(1000+i), 0)
	}

	// Rebuild FTS index
	if err := db.RebuildFTS(); err != nil {
		t.Fatalf("RebuildFTS: %v", err)
	}

	// Search for "networking" — should find 9 messages (all except msg-5)
	results, err := db.Search("networking", 50)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 9 {
		t.Errorf("expected 9 results for 'networking', got %d", len(results))
	}

	// Search for "firewalls" — should find 1 message
	results, err = db.Search("firewalls", 50)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'firewalls', got %d", len(results))
	}
	if len(results) > 0 {
		if results[0].MessageID != "msg-5" {
			t.Errorf("expected msg-5, got %s", results[0].MessageID)
		}
		if results[0].SenderName != "Alice" {
			t.Errorf("expected sender Alice, got %q", results[0].SenderName)
		}
	}
}

func TestSearchIndexMessage(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	insertTestConversation(t, db, "conv-1")
	insertTestParticipant(t, db, "part-1", "conv-1", "Bob")
	insertTestMessage(t, db, "msg-new", "conv-1", "part-1", "kubernetes cluster deployment", 2000, 1)

	// Index the single message
	if err := db.IndexMessage("msg-new", "kubernetes cluster deployment"); err != nil {
		t.Fatalf("IndexMessage: %v", err)
	}

	// Search for it
	results, err := db.Search("kubernetes", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].MessageID != "msg-new" {
		t.Errorf("expected msg-new, got %s", results[0].MessageID)
	}
	if !results[0].IsFromMe {
		t.Error("expected IsFromMe=true")
	}
	if results[0].SenderName != "Bob" {
		t.Errorf("expected sender Bob, got %q", results[0].SenderName)
	}
}

func TestSearchNoResults(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	insertTestConversation(t, db, "conv-1")
	insertTestMessage(t, db, "msg-1", "conv-1", "", "hello world", 1000, 0)

	if err := db.RebuildFTS(); err != nil {
		t.Fatalf("RebuildFTS: %v", err)
	}

	results, err := db.Search("xyznonexistent", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	results, err := db.Search("", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for empty query, got %v", results)
	}
}

func TestSearchRemoveFromFTS(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	insertTestConversation(t, db, "conv-1")
	insertTestMessage(t, db, "msg-del", "conv-1", "", "unique removable content", 1000, 0)

	if err := db.IndexMessage("msg-del", "unique removable content"); err != nil {
		t.Fatalf("IndexMessage: %v", err)
	}

	// Verify it's searchable
	results, err := db.Search("removable", 10)
	if err != nil {
		t.Fatalf("Search before remove: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result before remove, got %d", len(results))
	}

	// Remove from FTS
	if err := db.RemoveFromFTS("msg-del"); err != nil {
		t.Fatalf("RemoveFromFTS: %v", err)
	}

	// Verify no longer searchable via FTS
	// Note: the message still exists in messages table, but not in FTS index.
	// We verify by checking FTS directly.
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM messages_fts WHERE message_id = ?`, "msg-del").Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 FTS entries after remove, got %d", count)
	}
}

func TestSearchIndexMessageEmptyBody(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Indexing empty body should be a no-op
	if err := db.IndexMessage("msg-empty", ""); err != nil {
		t.Fatalf("IndexMessage empty: %v", err)
	}
	if err := db.IndexMessage("msg-spaces", "   "); err != nil {
		t.Fatalf("IndexMessage spaces: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM messages_fts`).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 FTS entries for empty bodies, got %d", count)
	}
}

func TestSearchLimit(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	insertTestConversation(t, db, "conv-1")
	for i := 0; i < 20; i++ {
		insertTestMessage(t, db, fmt.Sprintf("msg-%d", i), "conv-1", "", "repeated search term", int64(1000+i), 0)
	}
	if err := db.RebuildFTS(); err != nil {
		t.Fatalf("RebuildFTS: %v", err)
	}

	results, err := db.Search("repeated", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results with limit, got %d", len(results))
	}

	// Verify ordered by timestamp DESC (newest first)
	for i := 1; i < len(results); i++ {
		if results[i].TimestampMS > results[i-1].TimestampMS {
			t.Errorf("results not in DESC order: %d > %d", results[i].TimestampMS, results[i-1].TimestampMS)
		}
	}
}
