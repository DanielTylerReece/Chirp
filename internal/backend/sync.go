package backend

import (
	"database/sql"
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
	log.Println("sync: starting shallow backfill")

	convs, err := se.client.ListConversations(100)
	if err != nil {
		return fmt.Errorf("list conversations: %w", err)
	}

	log.Printf("sync: got %d conversations", len(convs))

	for _, c := range convs {
		dbConv := &db.Conversation{
			ID:                 c.ID,
			Name:               c.Name,
			IsGroup:            c.IsGroup,
			LastMessageTS:      c.LastMessageTS,
			LastMessagePreview: c.LastMessagePreview,
			IsPinned:           c.IsPinned,
			IsArchived:         c.IsArchived,
			IsRCS:              c.IsRCS,
			DefaultOutgoingID:  c.DefaultOutgoingID,
		}
		if c.Unread {
			dbConv.UnreadCount = 1 // We don't know exact count; flag as unread
		}

		if err := se.db.UpsertConversation(dbConv); err != nil {
			log.Printf("sync: upsert conversation %s: %v", c.ID, err)
			continue
		}

		// Upsert participants
		for _, p := range c.Participants {
			participant := &db.Participant{
				ID:             p.ID,
				ConversationID: c.ID,
				Name:           p.Name,
				PhoneNumber:    p.PhoneNumber,
				IsMe:           p.IsMe,
				AvatarHexColor: p.AvatarHexColor,
			}
			if p.ContactID != "" {
				participant.ContactID = sql.NullString{String: p.ContactID, Valid: true}
			}
			if err := se.db.UpsertParticipant(participant); err != nil {
				// FK errors are expected for group chat participants
				// whose conversation_id references aren't in our DB yet
			}
		}
	}

	// Emit a single event so the UI refreshes the conversation list
	se.bus.PublishConversation(app.ConversationEvent{ConversationID: "all"})

	log.Println("sync: shallow backfill complete")
	return nil
}

// BackfillEmptyConversations fetches messages for conversations that have none.
func (se *SyncEngine) BackfillEmptyConversations() error {
	ids, err := se.db.ConversationIDsWithoutMessages()
	if err != nil {
		return fmt.Errorf("list empty conversations: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}
	log.Printf("sync: backfilling messages for %d empty conversations", len(ids))

	filled := 0
	for _, convID := range ids {
		msgs, err := se.client.FetchMessages(convID, 50) // light backfill for empty conversations
		if err != nil {
			log.Printf("sync: fetch messages for %s: %v", convID, err)
			continue
		}
		for _, m := range msgs {
			dbMsg := &db.Message{
				ID:              m.ID,
				ConversationID:  m.ConversationID,
				ParticipantID:   m.ParticipantID,
				Body:            m.Body,
				TimestampMS:     m.TimestampMS,
				IsFromMe:        m.IsFromMe,
				Status:          m.Status,
				MediaID:         m.MediaID,
				MediaMimeType:   m.MediaMimeType,
				MediaDecryptKey: m.MediaDecryptKey,
				MediaSize:       m.MediaSize,
				MediaWidth:      m.MediaWidth,
				MediaHeight:     m.MediaHeight,
				ThumbnailID:     m.ThumbnailID,
				ThumbnailKey:    m.ThumbnailKey,
			}
			se.db.UpsertMessage(dbMsg)
		}
		if len(msgs) > 0 {
			filled++
		}
	}
	log.Printf("sync: backfilled %d/%d empty conversations", filled, len(ids))
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
	convs, err := se.client.ListConversations(1000)
	if err != nil {
		return fmt.Errorf("deep backfill conversations: %w", err)
	}

	for _, c := range convs {
		dbConv := &db.Conversation{
			ID:                 c.ID,
			Name:               c.Name,
			IsGroup:            c.IsGroup,
			LastMessageTS:      c.LastMessageTS,
			LastMessagePreview: c.LastMessagePreview,
			IsPinned:           c.IsPinned,
			IsArchived:         c.IsArchived,
			IsRCS:              c.IsRCS,
			DefaultOutgoingID:  c.DefaultOutgoingID,
		}
		if c.Unread {
			dbConv.UnreadCount = 1
		}
		if err := se.db.UpsertConversation(dbConv); err != nil {
			log.Printf("sync: deep upsert conversation %s: %v", c.ID, err)
			continue
		}

		for _, p := range c.Participants {
			participant := &db.Participant{
				ID:             p.ID,
				ConversationID: c.ID,
				Name:           p.Name,
				PhoneNumber:    p.PhoneNumber,
				IsMe:           p.IsMe,
				AvatarHexColor: p.AvatarHexColor,
			}
			if p.ContactID != "" {
				participant.ContactID = sql.NullString{String: p.ContactID, Valid: true}
			}
			if err := se.db.UpsertParticipant(participant); err != nil {
				log.Printf("sync: deep upsert participant %s: %v", p.ID, err)
			}
		}

		// Fetch messages for each conversation
		msgs, err := se.client.FetchMessages(c.ID, 800)
		if err != nil {
			log.Printf("sync: deep fetch messages for %s: %v", c.ID, err)
			continue
		}
		for _, m := range msgs {
			dbMsg := &db.Message{
				ID:             m.ID,
				ConversationID: m.ConversationID,
				ParticipantID:  m.ParticipantID,
				Body:           m.Body,
				TimestampMS:    m.TimestampMS,
				IsFromMe:       m.IsFromMe,
				Status:         m.Status,
				MediaID:        m.MediaID,
				MediaMimeType:  m.MediaMimeType,
				MediaSize:      m.MediaSize,
				MediaWidth:     m.MediaWidth,
				MediaHeight:    m.MediaHeight,
				ThumbnailID:    m.ThumbnailID,
				ReplyToID:      m.ReplyToID,
			}
			if err := se.db.UpsertMessage(dbMsg); err != nil {
				log.Printf("sync: deep upsert message %s: %v", m.ID, err)
			}
		}
	}

	// Phase 2: Fetch all contacts
	if err := se.client.ListContacts(); err != nil {
		log.Printf("sync: deep backfill contacts: %v", err)
		// Non-fatal, continue
	}

	se.bus.PublishConversation(app.ConversationEvent{ConversationID: "all"})

	log.Println("sync: deep backfill complete")
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
