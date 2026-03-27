package db

import "fmt"

type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		sql: `
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
`,
	},
	{
		version: 2,
		sql:     `ALTER TABLE conversations ADD COLUMN default_outgoing_id TEXT NOT NULL DEFAULT '';`,
	},
	{
		version: 3,
		sql:     `ALTER TABLE participants ADD COLUMN avatar_path TEXT NOT NULL DEFAULT '';`,
	},
	{
		version: 4,
		sql: `
UPDATE messages SET timestamp_ms = timestamp_ms / 1000 WHERE timestamp_ms > 1000000000000000;
UPDATE conversations SET last_message_ts = last_message_ts / 1000 WHERE last_message_ts > 1000000000000000;
`,
	},
}

func (db *DB) migrate() error {
	// Create schema_version if not exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY)`)
	if err != nil {
		return err
	}

	// Get current version
	var currentVersion int
	row := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&currentVersion); err != nil {
		return err
	}

	// Apply pending migrations
	for _, m := range migrations {
		if m.version > currentVersion {
			if _, err := db.Exec(m.sql); err != nil {
				return fmt.Errorf("migration %d: %w", m.version, err)
			}
			if _, err := db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, m.version); err != nil {
				return err
			}
		}
	}
	return nil
}
