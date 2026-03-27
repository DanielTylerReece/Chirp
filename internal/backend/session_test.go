package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionManagerFileFallback(t *testing.T) {
	// Use a temp directory for file-based fallback
	tmpDir := t.TempDir()

	sm := &SessionManager{
		fallbackPath: filepath.Join(tmpDir, "session.json"),
		useKeyring:   false, // Force file fallback
	}

	// No session initially
	if sm.HasSession() {
		t.Error("expected no session initially")
	}

	data, err := sm.Load()
	if err != nil {
		t.Fatalf("Load on empty: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data, got %s", data)
	}

	// Save session
	testData := []byte(`{"test": "auth_data", "token": "abc123"}`)
	if err := sm.Save(testData); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(filepath.Join(tmpDir, "session.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	// Load it back
	if !sm.HasSession() {
		t.Error("expected session after save")
	}

	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(loaded) != string(testData) {
		t.Errorf("data mismatch: got %s, want %s", loaded, testData)
	}

	// Clear
	if err := sm.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if sm.HasSession() {
		t.Error("expected no session after clear")
	}
}
