package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds XDG-compliant directory paths and application settings.
type Config struct {
	DataDir   string // ~/.local/share/gmessage
	ConfigDir string // ~/.config/gmessage
	CacheDir  string // ~/.cache/gmessage
	AvatarDir string // DataDir/avatars
	MediaDir  string // DataDir/media
	DBPath    string // DataDir/gmessage.db
	LogLevel  string
}

// NewConfig creates a Config with XDG-compliant paths.
func NewConfig() *Config {
	dataDir := filepath.Join(xdgDataHome(), "gmessage")
	configDir := filepath.Join(xdgConfigHome(), "gmessage")
	cacheDir := filepath.Join(xdgCacheHome(), "gmessage")

	logLevel := os.Getenv("GMESSAGE_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return &Config{
		DataDir:   dataDir,
		ConfigDir: configDir,
		CacheDir:  cacheDir,
		AvatarDir: filepath.Join(dataDir, "avatars"),
		MediaDir:  filepath.Join(dataDir, "media"),
		DBPath:    filepath.Join(dataDir, "gmessage.db"),
		LogLevel:  logLevel,
	}
}

// EnsureDirs creates all required directories.
func (c *Config) EnsureDirs() error {
	dirs := []string{c.DataDir, c.ConfigDir, c.CacheDir, c.AvatarDir, c.MediaDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}

func xdgDataHome() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func xdgConfigHome() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func xdgCacheHome() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache")
}

// WindowState holds persisted window geometry.
type WindowState struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

// windowStatePath returns the path to the window state JSON file.
func (c *Config) windowStatePath() string {
	return filepath.Join(c.ConfigDir, "window-state.json")
}

// LoadWindowState reads the saved window state. Returns defaults if none exists.
func (c *Config) LoadWindowState() WindowState {
	defaults := WindowState{Width: 1000, Height: 700, Maximized: false}

	data, err := os.ReadFile(c.windowStatePath())
	if err != nil {
		return defaults
	}

	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return defaults
	}

	// Sanity check: don't restore absurd sizes
	if state.Width < 400 {
		state.Width = defaults.Width
	}
	if state.Height < 300 {
		state.Height = defaults.Height
	}

	return state
}

// SaveWindowState persists the window geometry to disk.
func (c *Config) SaveWindowState(state WindowState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(c.windowStatePath(), data, 0600)
}
