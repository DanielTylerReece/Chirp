package db

import (
	"database/sql"
	"fmt"
)

// Conversation represents a chat conversation (1:1 or group).
type Conversation struct {
	ID                 string
	Name               string
	IsGroup            bool
	LastMessageTS      int64
	LastMessagePreview string
	UnreadCount        int
	IsPinned           bool
	IsArchived         bool
	IsRCS              bool
	AvatarURL          string
	CreatedAt          int64
	Participants       []Participant
}

// Participant represents a member of a conversation.
type Participant struct {
	ID             string
	ConversationID string
	ContactID      sql.NullString
	Name           string
	PhoneNumber    string
	IsMe           bool
	AvatarHexColor string
}

// boolToInt converts a Go bool to SQLite integer (0/1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// UpsertConversation inserts or updates a conversation.
func (db *DB) UpsertConversation(conv *Conversation) error {
	_, err := db.Exec(`
		INSERT INTO conversations (id, name, is_group, last_message_ts, last_message_preview,
			unread_count, is_pinned, is_archived, is_rcs, avatar_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			is_group = excluded.is_group,
			last_message_ts = excluded.last_message_ts,
			last_message_preview = excluded.last_message_preview,
			unread_count = excluded.unread_count,
			is_pinned = excluded.is_pinned,
			is_archived = excluded.is_archived,
			is_rcs = excluded.is_rcs,
			avatar_url = excluded.avatar_url,
			created_at = excluded.created_at`,
		conv.ID, conv.Name, boolToInt(conv.IsGroup), conv.LastMessageTS, conv.LastMessagePreview,
		conv.UnreadCount, boolToInt(conv.IsPinned), boolToInt(conv.IsArchived),
		boolToInt(conv.IsRCS), conv.AvatarURL, conv.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert conversation %s: %w", conv.ID, err)
	}
	return nil
}

// GetConversation retrieves a conversation by ID, including its participants.
func (db *DB) GetConversation(id string) (*Conversation, error) {
	conv := &Conversation{}
	var isGroup, isPinned, isArchived, isRCS int
	err := db.QueryRow(`
		SELECT id, name, is_group, last_message_ts, last_message_preview,
			unread_count, is_pinned, is_archived, is_rcs, avatar_url, created_at
		FROM conversations WHERE id = ?`, id,
	).Scan(
		&conv.ID, &conv.Name, &isGroup, &conv.LastMessageTS, &conv.LastMessagePreview,
		&conv.UnreadCount, &isPinned, &isArchived, &isRCS, &conv.AvatarURL, &conv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", id, err)
	}
	conv.IsGroup = isGroup != 0
	conv.IsPinned = isPinned != 0
	conv.IsArchived = isArchived != 0
	conv.IsRCS = isRCS != 0

	participants, err := db.GetParticipants(id)
	if err != nil {
		return nil, fmt.Errorf("get participants for conversation %s: %w", id, err)
	}
	conv.Participants = participants
	return conv, nil
}

// ListConversations returns conversations sorted by pinned first, then by last_message_ts DESC.
func (db *DB) ListConversations(limit, offset int) ([]Conversation, error) {
	rows, err := db.Query(`
		SELECT id, name, is_group, last_message_ts, last_message_preview,
			unread_count, is_pinned, is_archived, is_rcs, avatar_url, created_at
		FROM conversations
		ORDER BY is_pinned DESC, last_message_ts DESC
		LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		var isGroup, isPinned, isArchived, isRCS int
		if err := rows.Scan(
			&c.ID, &c.Name, &isGroup, &c.LastMessageTS, &c.LastMessagePreview,
			&c.UnreadCount, &isPinned, &isArchived, &isRCS, &c.AvatarURL, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		c.IsGroup = isGroup != 0
		c.IsPinned = isPinned != 0
		c.IsArchived = isArchived != 0
		c.IsRCS = isRCS != 0
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

// UpdateConversationLastMessage updates the last message timestamp and preview.
func (db *DB) UpdateConversationLastMessage(id string, ts int64, preview string) error {
	res, err := db.Exec(`UPDATE conversations SET last_message_ts = ?, last_message_preview = ? WHERE id = ?`,
		ts, preview, id)
	if err != nil {
		return fmt.Errorf("update conversation last message %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("update conversation last message %s: %w", id, sql.ErrNoRows)
	}
	return nil
}

// UpdateConversationUnread updates the unread count for a conversation.
func (db *DB) UpdateConversationUnread(id string, count int) error {
	res, err := db.Exec(`UPDATE conversations SET unread_count = ? WHERE id = ?`, count, id)
	if err != nil {
		return fmt.Errorf("update conversation unread %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("update conversation unread %s: %w", id, sql.ErrNoRows)
	}
	return nil
}

// DeleteConversation removes a conversation and cascades to messages and participants.
func (db *DB) DeleteConversation(id string) error {
	res, err := db.Exec(`DELETE FROM conversations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete conversation %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("delete conversation %s: %w", id, sql.ErrNoRows)
	}
	return nil
}

// UpsertParticipant inserts or updates a participant.
func (db *DB) UpsertParticipant(p *Participant) error {
	_, err := db.Exec(`
		INSERT INTO participants (id, conversation_id, contact_id, name, phone_number, is_me, avatar_hex_color)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			conversation_id = excluded.conversation_id,
			contact_id = excluded.contact_id,
			name = excluded.name,
			phone_number = excluded.phone_number,
			is_me = excluded.is_me,
			avatar_hex_color = excluded.avatar_hex_color`,
		p.ID, p.ConversationID, p.ContactID, p.Name, p.PhoneNumber,
		boolToInt(p.IsMe), p.AvatarHexColor,
	)
	if err != nil {
		return fmt.Errorf("upsert participant %s: %w", p.ID, err)
	}
	return nil
}

// GetParticipants returns all participants for a conversation.
func (db *DB) GetParticipants(conversationID string) ([]Participant, error) {
	rows, err := db.Query(`
		SELECT id, conversation_id, contact_id, name, phone_number, is_me, avatar_hex_color
		FROM participants WHERE conversation_id = ?`, conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("get participants for %s: %w", conversationID, err)
	}
	defer rows.Close()

	var participants []Participant
	for rows.Next() {
		var p Participant
		var isMe int
		if err := rows.Scan(&p.ID, &p.ConversationID, &p.ContactID, &p.Name,
			&p.PhoneNumber, &isMe, &p.AvatarHexColor); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		p.IsMe = isMe != 0
		participants = append(participants, p)
	}
	return participants, rows.Err()
}
