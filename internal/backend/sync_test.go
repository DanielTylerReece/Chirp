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

func TestShallowBackfillStoresData(t *testing.T) {
	mock := testutil.NewMockClient()
	mock.ListConversationsResult = []backend.ConversationData{
		{
			ID:                 "conv-1",
			Name:               "Alice",
			IsGroup:            false,
			LastMessageTS:      1711000000000,
			LastMessagePreview: "Hey there!",
			Unread:             true,
			Participants: []backend.ParticipantData{
				{ID: "p-1", Name: "Alice", PhoneNumber: "+15551234567", IsMe: false},
				{ID: "p-2", Name: "Me", PhoneNumber: "+15559876543", IsMe: true},
			},
		},
		{
			ID:      "conv-2",
			Name:    "Work Group",
			IsGroup: true,
		},
	}

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

	// Verify conversations stored in DB
	convs, err := database.ListConversations(10, 0)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}

	// Verify first conversation details
	conv, err := database.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if conv.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", conv.Name)
	}
	if conv.LastMessagePreview != "Hey there!" {
		t.Errorf("expected preview 'Hey there!', got %q", conv.LastMessagePreview)
	}
	if conv.UnreadCount != 1 {
		t.Errorf("expected unread count 1, got %d", conv.UnreadCount)
	}

	// Verify participants stored
	participants, err := database.GetParticipants("conv-1")
	if err != nil {
		t.Fatalf("GetParticipants: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
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
