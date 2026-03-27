package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWindowState_DefaultsWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ConfigDir: dir}

	state := cfg.LoadWindowState()

	if state.Width != 1000 {
		t.Errorf("expected default width 1000, got %d", state.Width)
	}
	if state.Height != 700 {
		t.Errorf("expected default height 700, got %d", state.Height)
	}
	if state.Maximized {
		t.Error("expected default maximized=false")
	}
}

func TestSaveAndLoadWindowState(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ConfigDir: dir}

	want := WindowState{Width: 1200, Height: 800, Maximized: true}
	if err := cfg.SaveWindowState(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := cfg.LoadWindowState()
	if got.Width != want.Width || got.Height != want.Height || got.Maximized != want.Maximized {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadWindowState_SanitizesSmallValues(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ConfigDir: dir}

	// Write a state with absurdly small dimensions
	if err := os.WriteFile(
		filepath.Join(dir, "window-state.json"),
		[]byte(`{"width":50,"height":50,"maximized":false}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	state := cfg.LoadWindowState()
	if state.Width < 400 {
		t.Errorf("expected width >= 400 after sanitize, got %d", state.Width)
	}
	if state.Height < 300 {
		t.Errorf("expected height >= 300 after sanitize, got %d", state.Height)
	}
}

func TestLoadWindowState_DefaultsOnCorruptFile(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ConfigDir: dir}

	if err := os.WriteFile(
		filepath.Join(dir, "window-state.json"),
		[]byte(`not json`),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	state := cfg.LoadWindowState()
	if state.Width != 1000 || state.Height != 700 {
		t.Errorf("expected defaults on corrupt file, got %+v", state)
	}
}
