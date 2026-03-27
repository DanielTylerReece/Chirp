package db

import (
	"fmt"
	"strings"
)

// SearchResult holds a message matched by full-text search.
type SearchResult struct {
	MessageID      string
	ConversationID string
	Body           string
	TimestampMS    int64
	IsFromMe       bool
	SenderName     string // from participant join
}

// RebuildFTS rebuilds the FTS index from scratch. Call on startup.
func (db *DB) RebuildFTS() error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM messages_fts`); err != nil {
		return fmt.Errorf("clear fts: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO messages_fts(message_id, body) SELECT id, body FROM messages WHERE body != ''`); err != nil {
		return fmt.Errorf("rebuild fts: %w", err)
	}

	return tx.Commit()
}

// IndexMessage adds a single message to the FTS index. Call on new message.
func (db *DB) IndexMessage(id string, body string) error {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO messages_fts(message_id, body) VALUES (?, ?)`, id, body)
	if err != nil {
		return fmt.Errorf("index message: %w", err)
	}
	return nil
}

// Search performs a full-text search and returns matching messages.
// Falls back to LIKE if FTS fails.
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	// Try FTS first (trigram tokenizer — query is used directly, no special syntax)
	results, err := db.searchFTS(query, limit)
	if err == nil {
		return results, nil
	}

	// Fall back to LIKE
	return db.searchLIKE(query, limit)
}

func (db *DB) searchFTS(query string, limit int) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT m.id, m.conversation_id, m.body, m.timestamp_ms, m.is_from_me,
		       COALESCE(p.name, '') AS sender_name
		FROM messages_fts f
		JOIN messages m ON f.message_id = m.id
		LEFT JOIN participants p ON m.participant_id = p.id
		WHERE messages_fts MATCH ?
		ORDER BY m.timestamp_ms DESC
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fts query: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

func (db *DB) searchLIKE(query string, limit int) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT m.id, m.conversation_id, m.body, m.timestamp_ms, m.is_from_me,
		       COALESCE(p.name, '') AS sender_name
		FROM messages m
		LEFT JOIN participants p ON m.participant_id = p.id
		WHERE m.body LIKE '%' || ? || '%'
		ORDER BY m.timestamp_ms DESC
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("like query: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

func scanSearchResults(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var isFromMe int
		if err := rows.Scan(&r.MessageID, &r.ConversationID, &r.Body, &r.TimestampMS, &isFromMe, &r.SenderName); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		r.IsFromMe = isFromMe != 0
		results = append(results, r)
	}
	return results, rows.Err()
}

// RemoveFromFTS removes a message from the FTS index.
func (db *DB) RemoveFromFTS(id string) error {
	_, err := db.Exec(`DELETE FROM messages_fts WHERE message_id = ?`, id)
	if err != nil {
		return fmt.Errorf("remove from fts: %w", err)
	}
	return nil
}
