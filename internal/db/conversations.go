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
	DefaultOutgoingID  string // Participant ID of the "me" SIM for this conversation
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
	AvatarPath     string // Path to cached avatar JPEG on disk (empty if none)
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
			unread_count, is_pinned, is_archived, is_rcs, avatar_url, created_at, default_outgoing_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			created_at = excluded.created_at,
			default_outgoing_id = excluded.default_outgoing_id`,
		conv.ID, conv.Name, boolToInt(conv.IsGroup), conv.LastMessageTS, conv.LastMessagePreview,
		conv.UnreadCount, boolToInt(conv.IsPinned), boolToInt(conv.IsArchived),
		boolToInt(conv.IsRCS), conv.AvatarURL, conv.CreatedAt, conv.DefaultOutgoingID,
	)
	if err != nil {
		return fmt.Errorf("upsert conversation %s: %w", conv.ID, err)
	}
	return nil
}

// UpdateConversationPreview updates the last message preview and timestamp for a conversation.
func (db *DB) UpdateConversationPreview(convID string, preview string, timestampMS int64) error {
	_, err := db.Exec(`
		UPDATE conversations SET last_message_preview = ?, last_message_ts = ?
		WHERE id = ? AND last_message_ts <= ?`,
		preview, timestampMS, convID, timestampMS,
	)
	if err != nil {
		return fmt.Errorf("update conversation preview %s: %w", convID, err)
	}
	return nil
}

// GetConversation retrieves a conversation by ID, including its participants.
func (db *DB) GetConversation(id string) (*Conversation, error) {
	conv := &Conversation{}
	var isGroup, isPinned, isArchived, isRCS int
	err := db.QueryRow(`
		SELECT id, name, is_group, last_message_ts, last_message_preview,
			unread_count, is_pinned, is_archived, is_rcs, avatar_url, created_at, default_outgoing_id
		FROM conversations WHERE id = ?`, id,
	).Scan(
		&conv.ID, &conv.Name, &isGroup, &conv.LastMessageTS, &conv.LastMessagePreview,
		&conv.UnreadCount, &isPinned, &isArchived, &isRCS, &conv.AvatarURL, &conv.CreatedAt,
		&conv.DefaultOutgoingID,
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
// AvatarURL is populated with the first non-me participant's cached avatar path (if any).
func (db *DB) ListConversations(limit, offset int) ([]Conversation, error) {
	rows, err := db.Query(`
		SELECT c.id, c.name, c.is_group, c.last_message_ts, c.last_message_preview,
			c.unread_count, c.is_pinned, c.is_archived, c.is_rcs,
			COALESCE((
				SELECT p.avatar_path FROM participants p
				WHERE p.conversation_id = c.id AND p.is_me = 0 AND p.avatar_path != ''
				LIMIT 1
			), c.avatar_url) AS avatar_url,
			c.created_at, c.default_outgoing_id
		FROM conversations c
		ORDER BY c.is_pinned DESC, c.last_message_ts DESC
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
			&c.DefaultOutgoingID,
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
// Note: avatar_path is preserved on conflict — it is only set via UpdateParticipantAvatar.
func (db *DB) UpsertParticipant(p *Participant) error {
	_, err := db.Exec(`
		INSERT INTO participants (id, conversation_id, contact_id, name, phone_number, is_me, avatar_hex_color, avatar_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, '')
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

// ListParticipantIDsWithoutAvatar returns distinct participant IDs that are not
// "me" and don't have a cached avatar yet.
func (db *DB) ListParticipantIDsWithoutAvatar() ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT id FROM participants WHERE is_me = 0 AND avatar_path = ''`)
	if err != nil {
		return nil, fmt.Errorf("list participant IDs without avatar: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// UpdateParticipantAvatar sets the avatar file path for a participant.
func (db *DB) UpdateParticipantAvatar(participantID string, path string) error {
	_, err := db.Exec(`UPDATE participants SET avatar_path = ? WHERE id = ?`, path, participantID)
	if err != nil {
		return fmt.Errorf("update participant avatar %s: %w", participantID, err)
	}
	return nil
}

// GetParticipants returns all participants for a conversation.
func (db *DB) GetParticipants(conversationID string) ([]Participant, error) {
	rows, err := db.Query(`
		SELECT id, conversation_id, contact_id, name, phone_number, is_me, avatar_hex_color, avatar_path
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
			&p.PhoneNumber, &isMe, &p.AvatarHexColor, &p.AvatarPath); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		p.IsMe = isMe != 0
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// SearchConversations searches conversations by name.
func (db *DB) SearchConversations(query string) ([]Conversation, error) {
	like := "%" + query + "%"
	rows, err := db.Query(`
		SELECT id, name, last_message_preview, last_message_ts, unread_count,
			is_pinned, is_archived, is_rcs, avatar_url, created_at, default_outgoing_id, is_group
		FROM conversations
		WHERE name LIKE ?
		ORDER BY last_message_ts DESC
		LIMIT 20`, like,
	)
	if err != nil {
		return nil, fmt.Errorf("search conversations: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		var isPinned, isArchived, isRCS, isGroup int
		if err := rows.Scan(&c.ID, &c.Name, &c.LastMessagePreview, &c.LastMessageTS,
			&c.UnreadCount, &isPinned, &isArchived, &isRCS, &c.AvatarURL, &c.CreatedAt,
			&c.DefaultOutgoingID, &isGroup); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		c.IsPinned = isPinned != 0
		c.IsArchived = isArchived != 0
		c.IsRCS = isRCS != 0
		c.IsGroup = isGroup != 0
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

// SearchRecipients searches participants (excluding "me") by name or phone number.
// Returns unique results deduplicated by phone number.
func (db *DB) SearchRecipients(query string) ([]Participant, error) {
	like := "%" + query + "%"
	rows, err := db.Query(`
		SELECT DISTINCT p.id, p.conversation_id, p.contact_id, p.name,
			p.phone_number, p.is_me, p.avatar_hex_color, p.avatar_path
		FROM participants p
		WHERE p.is_me = 0
			AND p.phone_number <> ''
			AND (p.name LIKE ? OR p.phone_number LIKE ?)
		ORDER BY p.name
		LIMIT 20`, like, like,
	)
	if err != nil {
		return nil, fmt.Errorf("search recipients: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var results []Participant
	for rows.Next() {
		var p Participant
		var isMe int
		if err := rows.Scan(&p.ID, &p.ConversationID, &p.ContactID, &p.Name,
			&p.PhoneNumber, &isMe, &p.AvatarHexColor, &p.AvatarPath); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		p.IsMe = isMe != 0
		if !seen[p.PhoneNumber] {
			seen[p.PhoneNumber] = true
			results = append(results, p)
		}
	}
	return results, rows.Err()
}
