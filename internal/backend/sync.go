package backend

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/db"
)

// SyncEngine manages data synchronization between libgm and the local database.
type SyncEngine struct {
	client GMClient
	db     *db.DB
	bus    *app.EventBus
	config *app.Config

	backfillRunning atomic.Bool
	mu              sync.Mutex
}

func NewSyncEngine(client GMClient, database *db.DB, bus *app.EventBus, config *app.Config) *SyncEngine {
	return &SyncEngine{
		client: client,
		db:     database,
		bus:    bus,
		config: config,
	}
}

// ShallowBackfill fetches recent conversations and messages.
// Called on every startup after connection.
func (se *SyncEngine) ShallowBackfill() error {
	// 1. ListConversations(100) via client
	// 2. For each conversation: FetchMessages(convID, 50) via client
	// 3. Store in DB
	// 4. Emit events
	//
	// Note: Since our GMClient interface returns errors (not response objects),
	// the actual data comes through the event handler (WrappedMessage, Conversation events).
	// So ShallowBackfill just triggers the fetches — the EventRouter handles storage.

	log.Println("sync: starting shallow backfill")

	if err := se.client.ListConversations(100); err != nil {
		return fmt.Errorf("list conversations: %w", err)
	}

	// Messages will come through events
	log.Println("sync: shallow backfill triggered")
	return nil
}

// DeepBackfill fetches ALL conversations and messages.
// Runs in background, only one at a time.
func (se *SyncEngine) DeepBackfill() error {
	if !se.backfillRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("backfill already running")
	}
	defer se.backfillRunning.Store(false)

	log.Println("sync: starting deep backfill")

	// Phase 1: Fetch all conversations
	if err := se.client.ListConversations(1000); err != nil {
		return fmt.Errorf("deep backfill conversations: %w", err)
	}

	// Phase 2: Fetch all contacts
	if err := se.client.ListContacts(); err != nil {
		log.Printf("sync: deep backfill contacts: %v", err)
		// Non-fatal, continue
	}

	log.Println("sync: deep backfill triggered")
	return nil
}

// IsBackfilling returns true if a deep backfill is in progress.
func (se *SyncEngine) IsBackfilling() bool {
	return se.backfillRunning.Load()
}

// SyncContacts fetches contacts and their thumbnails.
func (se *SyncEngine) SyncContacts() error {
	log.Println("sync: fetching contacts")
	return se.client.ListContacts()
}
