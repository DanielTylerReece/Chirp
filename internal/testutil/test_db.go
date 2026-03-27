package testutil

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// NewTestDB creates an in-memory SQLite database with the full GMessage schema applied.
// The database is automatically closed when the test finishes.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	// Single connection — in-memory DBs are not shared across connections.
	db.SetMaxOpenConns(1)

	if err := applySchema(db); err != nil {
		db.Close()
		t.Fatalf("failed to apply schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func applySchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id               TEXT PRIMARY KEY,
		name             TEXT NOT NULL DEFAULT '',
		is_group         INTEGER NOT NULL DEFAULT 0,
		last_message_ts  INTEGER NOT NULL DEFAULT 0,
		last_message_preview TEXT NOT NULL DEFAULT '',
		unread_count     INTEGER NOT NULL DEFAULT 0,
		is_pinned        INTEGER NOT NULL DEFAULT 0,
		is_archived      INTEGER NOT NULL DEFAULT 0,
		is_rcs           INTEGER NOT NULL DEFAULT 0,
		avatar_url       TEXT NOT NULL DEFAULT '',
		created_at       INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_conv_last_msg ON conversations(last_message_ts DESC);

	CREATE TABLE IF NOT EXISTS contacts (
		id               TEXT PRIMARY KEY,
		name             TEXT NOT NULL DEFAULT '',
		phone_number     TEXT NOT NULL DEFAULT '',
		avatar_cached    INTEGER NOT NULL DEFAULT 0,
		avatar_path      TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_contact_phone ON contacts(phone_number);

	CREATE TABLE IF NOT EXISTS participants (
		id               TEXT PRIMARY KEY,
		conversation_id  TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		contact_id       TEXT DEFAULT NULL REFERENCES contacts(id),
		name             TEXT NOT NULL DEFAULT '',
		phone_number     TEXT NOT NULL DEFAULT '',
		is_me            INTEGER NOT NULL DEFAULT 0,
		avatar_hex_color TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_part_conv ON participants(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_part_phone ON participants(phone_number);

	CREATE TABLE IF NOT EXISTS messages (
		id               TEXT PRIMARY KEY,
		conversation_id  TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		participant_id   TEXT NOT NULL DEFAULT '',
		body             TEXT NOT NULL DEFAULT '',
		timestamp_ms     INTEGER NOT NULL DEFAULT 0,
		is_from_me       INTEGER NOT NULL DEFAULT 0,
		status           INTEGER NOT NULL DEFAULT 0,
		media_id         TEXT NOT NULL DEFAULT '',
		media_mime_type  TEXT NOT NULL DEFAULT '',
		media_decrypt_key BLOB DEFAULT NULL,
		media_size       INTEGER NOT NULL DEFAULT 0,
		media_width      INTEGER NOT NULL DEFAULT 0,
		media_height     INTEGER NOT NULL DEFAULT 0,
		thumbnail_id     TEXT NOT NULL DEFAULT '',
		thumbnail_key    BLOB DEFAULT NULL,
		reply_to_id      TEXT NOT NULL DEFAULT '',
		reactions        TEXT NOT NULL DEFAULT '[]'
	);
	CREATE INDEX IF NOT EXISTS idx_msg_conv_ts ON messages(conversation_id, timestamp_ms DESC);
	CREATE INDEX IF NOT EXISTS idx_msg_ts ON messages(timestamp_ms DESC);

	CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		message_id UNINDEXED,
		body,
		tokenize='trigram'
	);

	CREATE TABLE IF NOT EXISTS media_cache (
		media_id         TEXT PRIMARY KEY,
		local_path       TEXT NOT NULL,
		mime_type        TEXT NOT NULL DEFAULT '',
		cached_at        INTEGER NOT NULL DEFAULT 0,
		size_bytes       INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY
	);
	INSERT OR IGNORE INTO schema_version (version) VALUES (1);
	`
	_, err := db.Exec(schema)
	return err
}
