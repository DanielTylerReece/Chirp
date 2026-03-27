package backend_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/backend"
	"github.com/tyler/gmessage/internal/db"
	"github.com/tyler/gmessage/internal/testutil"
)

func TestResolveParticipantName(t *testing.T) {
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	mock := testutil.NewMockClient()
	config := app.NewConfig()
	cm := backend.NewContactManager(mock, database, config)

	// No contact — should return formatted phone
	name := cm.ResolveParticipantName("+15551234567")
	if name != "+15551234567" {
		t.Errorf("expected phone number, got %q", name)
	}

	// Add contact
	database.UpsertContact(&db.Contact{
		ID:          "c1",
		Name:        "Alice Smith",
		PhoneNumber: "+15551234567",
	})

	// Now should resolve
	name = cm.ResolveParticipantName("+15551234567")
	if name != "Alice Smith" {
		t.Errorf("expected 'Alice Smith', got %q", name)
	}
}

func TestCacheAvatar(t *testing.T) {
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	mock := testutil.NewMockClient()
	tmpDir := t.TempDir()
	config := &app.Config{
		AvatarDir: filepath.Join(tmpDir, "avatars"),
	}
	os.MkdirAll(config.AvatarDir, 0700)

	cm := backend.NewContactManager(mock, database, config)

	// Add contact first
	database.UpsertContact(&db.Contact{
		ID:          "c1",
		Name:        "Alice",
		PhoneNumber: "+15551234567",
	})

	// Cache avatar
	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes
	err = cm.CacheAvatar("c1", fakeJPEG)
	if err != nil {
		t.Fatalf("CacheAvatar: %v", err)
	}

	// Verify file exists
	path := cm.GetAvatarPath("c1")
	if path == "" {
		t.Error("expected non-empty avatar path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read avatar: %v", err)
	}
	if len(data) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(data))
	}
}

func TestNormalizePhone(t *testing.T) {
	// Test via ResolveParticipantName matching
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	database.UpsertContact(&db.Contact{
		ID:          "c1",
		Name:        "Bob",
		PhoneNumber: "+15559876543",
	})

	mock := testutil.NewMockClient()
	config := app.NewConfig()
	cm := backend.NewContactManager(mock, database, config)

	// Exact match
	name := cm.ResolveParticipantName("+15559876543")
	if name != "Bob" {
		t.Errorf("exact match failed: got %q", name)
	}
}
