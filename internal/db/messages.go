package db

import (
	"database/sql"
	"fmt"
)

// Message represents a single message in a conversation.
type Message struct {
	ID              string
	ConversationID  string
	ParticipantID   string
	Body            string
	TimestampMS     int64
	IsFromMe        bool
	Status          int // 0=sending, 1=sent, 2=delivered, 3=read, 4=failed
	MediaID         string
	MediaMimeType   string
	MediaDecryptKey []byte
	MediaSize       int64
	MediaWidth      int
	MediaHeight     int
	ThumbnailID     string
	ThumbnailKey    []byte
	ReplyToID       string
	Reactions       string // JSON
}

// UpsertMessage inserts or updates a message.
func (db *DB) UpsertMessage(msg *Message) error {
	_, err := db.Exec(`
		INSERT INTO messages (id, conversation_id, participant_id, body, timestamp_ms,
			is_from_me, status, media_id, media_mime_type, media_decrypt_key,
			media_size, media_width, media_height, thumbnail_id, thumbnail_key,
			reply_to_id, reactions)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			conversation_id = excluded.conversation_id,
			participant_id = excluded.participant_id,
			body = excluded.body,
			timestamp_ms = excluded.timestamp_ms,
			is_from_me = excluded.is_from_me,
			status = excluded.status,
			media_id = excluded.media_id,
			media_mime_type = excluded.media_mime_type,
			media_decrypt_key = excluded.media_decrypt_key,
			media_size = excluded.media_size,
			media_width = excluded.media_width,
			media_height = excluded.media_height,
			thumbnail_id = excluded.thumbnail_id,
			thumbnail_key = excluded.thumbnail_key,
			reply_to_id = excluded.reply_to_id,
			reactions = excluded.reactions`,
		msg.ID, msg.ConversationID, msg.ParticipantID, msg.Body, msg.TimestampMS,
		boolToInt(msg.IsFromMe), msg.Status, msg.MediaID, msg.MediaMimeType,
		msg.MediaDecryptKey, msg.MediaSize, msg.MediaWidth, msg.MediaHeight,
		msg.ThumbnailID, msg.ThumbnailKey, msg.ReplyToID, msg.Reactions,
	)
	if err != nil {
		return fmt.Errorf("upsert message %s: %w", msg.ID, err)
	}
	return nil
}

// GetMessage retrieves a single message by ID.
func (db *DB) GetMessage(id string) (*Message, error) {
	msg := &Message{}
	var isFromMe int
	err := db.QueryRow(`
		SELECT id, conversation_id, participant_id, body, timestamp_ms,
			is_from_me, status, media_id, media_mime_type, media_decrypt_key,
			media_size, media_width, media_height, thumbnail_id, thumbnail_key,
			reply_to_id, reactions
		FROM messages WHERE id = ?`, id,
	).Scan(
		&msg.ID, &msg.ConversationID, &msg.ParticipantID, &msg.Body, &msg.TimestampMS,
		&isFromMe, &msg.Status, &msg.MediaID, &msg.MediaMimeType, &msg.MediaDecryptKey,
		&msg.MediaSize, &msg.MediaWidth, &msg.MediaHeight, &msg.ThumbnailID,
		&msg.ThumbnailKey, &msg.ReplyToID, &msg.Reactions,
	)
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", id, err)
	}
	msg.IsFromMe = isFromMe != 0
	return msg, nil
}

// GetMessages returns messages for a conversation ordered by timestamp DESC.
// If beforeTimestamp is 0, returns the latest messages. Otherwise filters WHERE timestamp_ms < beforeTimestamp.
func (db *DB) GetMessages(conversationID string, limit int, beforeTimestamp int64) ([]Message, error) {
	var rows *sql.Rows
	var err error

	if beforeTimestamp == 0 {
		rows, err = db.Query(`
			SELECT id, conversation_id, participant_id, body, timestamp_ms,
				is_from_me, status, media_id, media_mime_type, media_decrypt_key,
				media_size, media_width, media_height, thumbnail_id, thumbnail_key,
				reply_to_id, reactions
			FROM (
				SELECT * FROM messages
				WHERE conversation_id = ?
				ORDER BY timestamp_ms DESC
				LIMIT ?
			) sub ORDER BY timestamp_ms ASC`, conversationID, limit,
		)
	} else {
		rows, err = db.Query(`
			SELECT id, conversation_id, participant_id, body, timestamp_ms,
				is_from_me, status, media_id, media_mime_type, media_decrypt_key,
				media_size, media_width, media_height, thumbnail_id, thumbnail_key,
				reply_to_id, reactions
			FROM (
				SELECT * FROM messages
				WHERE conversation_id = ? AND timestamp_ms < ?
				ORDER BY timestamp_ms DESC
				LIMIT ?
			) sub ORDER BY timestamp_ms ASC`, conversationID, beforeTimestamp, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("get messages for %s: %w", conversationID, err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var isFromMe int
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.ParticipantID, &m.Body, &m.TimestampMS,
			&isFromMe, &m.Status, &m.MediaID, &m.MediaMimeType, &m.MediaDecryptKey,
			&m.MediaSize, &m.MediaWidth, &m.MediaHeight, &m.ThumbnailID,
			&m.ThumbnailKey, &m.ReplyToID, &m.Reactions,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.IsFromMe = isFromMe != 0
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// UpdateMessageStatus updates the delivery status of a message.
func (db *DB) UpdateMessageStatus(id string, status int) error {
	res, err := db.Exec(`UPDATE messages SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update message status %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("update message status %s: %w", id, sql.ErrNoRows)
	}
	return nil
}

// DeleteMessage removes a message by ID.
func (db *DB) DeleteMessage(id string) error {
	res, err := db.Exec(`DELETE FROM messages WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete message %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("delete message %s: %w", id, sql.ErrNoRows)
	}
	return nil
}

// ConversationIDsWithoutMessages returns IDs of conversations that have no messages.
func (db *DB) ConversationIDsWithoutMessages() ([]string, error) {
	rows, err := db.Query(`
		SELECT c.id FROM conversations c
		LEFT JOIN messages m ON m.conversation_id = c.id
		GROUP BY c.id
		HAVING COUNT(m.id) = 0`)
	if err != nil {
		return nil, fmt.Errorf("conversation IDs without messages: %w", err)
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

// CountMessages returns the total number of messages in a conversation.
func (db *DB) CountMessages(conversationID string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, conversationID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count messages for %s: %w", conversationID, err)
	}
	return count, nil
}
