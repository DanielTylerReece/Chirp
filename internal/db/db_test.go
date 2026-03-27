package db

import (
	"testing"
)

func TestOpenMemory(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Verify all tables exist
	tables := []string{"conversations", "contacts", "participants", "messages", "media_cache", "schema_version"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}

	// Verify FTS table
	var ftsName string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='messages_fts'`).Scan(&ftsName)
	if err != nil {
		t.Errorf("FTS table not found: %v", err)
	}

	// Verify schema version
	var version int
	err = db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version)
	if err != nil {
		t.Fatalf("schema version query: %v", err)
	}
	if version != 1 {
		t.Errorf("expected schema version 1, got %d", version)
	}

	// Verify WAL mode
	var journalMode string
	err = db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode)
	if err != nil {
		t.Fatalf("journal_mode query: %v", err)
	}
	if journalMode != "wal" {
		// In-memory databases may not support WAL, that's ok
		t.Logf("journal_mode: %s (memory databases may not use wal)", journalMode)
	}

	// Verify foreign keys enabled
	var fk int
	err = db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk)
	if err != nil {
		t.Fatalf("foreign_keys query: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys not enabled: %d", fk)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Run migrate again — should be a no-op
	if err := db.migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	// Verify still version 1
	var version int
	db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version)
	if version != 1 {
		t.Errorf("expected schema version 1 after re-migrate, got %d", version)
	}
}

func TestForeignKeyConstraint(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer db.Close()

	// Insert message with non-existent conversation should fail
	_, err = db.Exec(`INSERT INTO messages (id, conversation_id, body) VALUES ('msg-1', 'nonexistent', 'hello')`)
	if err == nil {
		t.Error("expected foreign key constraint error, got nil")
	}
}
