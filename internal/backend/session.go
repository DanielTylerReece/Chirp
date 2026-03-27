package backend

import (
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "com.github.gmessage"
	keyringUser    = "session"
)

// SessionManager handles persisting and restoring libgm auth data.
// Primary storage: GNOME Keyring via libsecret.
// Fallback: file-based storage with 0600 permissions (for headless/no-keyring environments).
type SessionManager struct {
	fallbackPath string // e.g., ~/.local/share/gmessage/session.json
	useKeyring   bool
}

func NewSessionManager(dataDir string) *SessionManager {
	sm := &SessionManager{
		fallbackPath: filepath.Join(dataDir, "session.json"),
		useKeyring:   true,
	}
	// Test if keyring is available
	if err := keyring.Set(keyringService, "test", "test"); err != nil {
		sm.useKeyring = false
	} else {
		keyring.Delete(keyringService, "test")
	}
	return sm
}

// Save persists the auth data JSON string.
func (sm *SessionManager) Save(authDataJSON []byte) error {
	if sm.useKeyring {
		if err := keyring.Set(keyringService, keyringUser, string(authDataJSON)); err != nil {
			// Fall back to file
			return sm.saveToFile(authDataJSON)
		}
		return nil
	}
	return sm.saveToFile(authDataJSON)
}

// Load retrieves the auth data JSON string.
// Returns nil, nil if no session exists.
func (sm *SessionManager) Load() ([]byte, error) {
	if sm.useKeyring {
		secret, err := keyring.Get(keyringService, keyringUser)
		if err == keyring.ErrNotFound {
			return sm.loadFromFile()
		}
		if err != nil {
			return sm.loadFromFile()
		}
		return []byte(secret), nil
	}
	return sm.loadFromFile()
}

// Clear removes the stored session.
func (sm *SessionManager) Clear() error {
	if sm.useKeyring {
		keyring.Delete(keyringService, keyringUser)
	}
	os.Remove(sm.fallbackPath)
	return nil
}

// HasSession returns true if a session exists in either storage.
func (sm *SessionManager) HasSession() bool {
	data, _ := sm.Load()
	return len(data) > 0
}

func (sm *SessionManager) saveToFile(data []byte) error {
	dir := filepath.Dir(sm.fallbackPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(sm.fallbackPath, data, 0600)
}

func (sm *SessionManager) loadFromFile() ([]byte, error) {
	data, err := os.ReadFile(sm.fallbackPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}
