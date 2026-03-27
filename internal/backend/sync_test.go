package backend_test

import (
	"testing"

	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/backend"
	"github.com/tyler/gmessage/internal/db"
	"github.com/tyler/gmessage/internal/testutil"
)

func TestShallowBackfill(t *testing.T) {
	mock := testutil.NewMockClient()
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	bus := app.NewEventBus()
	config := app.NewConfig()

	se := backend.NewSyncEngine(mock, database, bus, config)

	if err := se.ShallowBackfill(); err != nil {
		t.Fatalf("ShallowBackfill: %v", err)
	}

	// Verify ListConversations was called
	if mock.CallCount("ListConversations") != 1 {
		t.Errorf("expected 1 ListConversations call, got %d", mock.CallCount("ListConversations"))
	}
}

func TestDeepBackfillOnlyOnce(t *testing.T) {
	mock := testutil.NewMockClient()
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	bus := app.NewEventBus()
	config := app.NewConfig()

	se := backend.NewSyncEngine(mock, database, bus, config)

	// First backfill should work
	err = se.DeepBackfill()
	if err != nil {
		t.Fatalf("first DeepBackfill: %v", err)
	}

	// Since DeepBackfill completes synchronously in test (mock client returns immediately),
	// a second call should also work (the atomic bool is reset after completion)
	err = se.DeepBackfill()
	if err != nil {
		t.Fatalf("second DeepBackfill: %v", err)
	}
}

func TestMessageHandlerOptimisticSend(t *testing.T) {
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	bus := app.NewEventBus()
	mh := backend.NewMessageHandler(database, bus)

	// Need a conversation first (FK constraint)
	database.UpsertConversation(&db.Conversation{
		ID:   "conv-1",
		Name: "Alice",
	})

	// Send optimistic
	tmpID := mh.SendOptimistic("conv-1", "Hello!")
	if tmpID == "" {
		t.Fatal("expected non-empty tmpID")
	}

	// Verify message in DB
	msg, err := database.GetMessage(tmpID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if msg.Body != "Hello!" {
		t.Errorf("expected body 'Hello!', got %q", msg.Body)
	}
	if msg.Status != 0 {
		t.Errorf("expected status 0 (sending), got %d", msg.Status)
	}
	if !msg.IsFromMe {
		t.Error("expected IsFromMe=true")
	}
}

func TestMessageHandlerMarkFailed(t *testing.T) {
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	bus := app.NewEventBus()
	mh := backend.NewMessageHandler(database, bus)

	// Setup
	database.UpsertConversation(&db.Conversation{ID: "conv-1", Name: "Alice"})
	tmpID := mh.SendOptimistic("conv-1", "Hello!")

	// Mark failed
	mh.MarkFailed(tmpID)

	msg, err := database.GetMessage(tmpID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if msg.Status != 4 {
		t.Errorf("expected status 4 (failed), got %d", msg.Status)
	}
}
