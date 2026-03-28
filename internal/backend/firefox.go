package backend

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// ReadFirefoxCookies reads Google cookies from Firefox or Zen Browser's cookie database.
// Returns the required cookies or an error if they can't be found.
func ReadFirefoxCookies() (map[string]string, error) {
	profileDir, err := findFirefoxProfile()
	if err != nil {
		return nil, err
	}
	return readCookiesFromProfile(profileDir)
}

func readCookiesFromProfile(profileDir string) (map[string]string, error) {

	cookiesPath := filepath.Join(profileDir, "cookies.sqlite")
	if _, err := os.Stat(cookiesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cookies.sqlite not found in Firefox profile")
	}

	// Open read-only to avoid locking issues while Firefox is running
	db, err := sql.Open("sqlite", cookiesPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open Firefox cookies: %w", err)
	}
	defer db.Close()

	// Query ALL Google cookies, prioritizing messages.google.com then .google.com
	rows, err := db.Query(`
		SELECT name, value, host FROM moz_cookies
		WHERE host LIKE '%google.com'
		AND value != ''
		ORDER BY
			CASE WHEN host = 'messages.google.com' THEN 0
			     WHEN host = '.google.com' THEN 1
			     ELSE 2
			END,
			expiry DESC`)
	if err != nil {
		return nil, fmt.Errorf("query Firefox cookies: %w (is Firefox closed?)", err)
	}
	defer rows.Close()

	cookies := make(map[string]string)
	for rows.Next() {
		var name, value, host string
		if err := rows.Scan(&name, &value, &host); err != nil {
			continue
		}
		_ = host
		// Keep first match per cookie name (ordered by host priority above)
		if cookies[name] == "" {
			cookies[name] = value
		}
	}

	// Validate
	var missing []string
	for _, name := range requiredCookies {
		if cookies[name] == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing cookies in Firefox: %s\nMake sure you're signed into messages.google.com", strings.Join(missing, ", "))
	}

	return cookies, nil
}

// findFirefoxProfile finds the default Firefox or Zen Browser profile directory.
func findFirefoxProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Search order: Zen Browser, then Firefox
	searchDirs := []string{
		filepath.Join(home, ".config", "zen"),                                    // Zen Browser
		filepath.Join(home, ".mozilla", "firefox"),                               // Firefox
		filepath.Join(home, "snap", "firefox", "common", ".mozilla", "firefox"),  // Firefox Snap
	}

	for _, dir := range searchDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// Try profiles.ini for the default profile
		iniPath := filepath.Join(dir, "profiles.ini")
		if data, err := os.ReadFile(iniPath); err == nil {
			defaultPath := parseDefaultProfile(string(data), dir)
			if defaultPath != "" {
				if _, err := os.Stat(filepath.Join(defaultPath, "cookies.sqlite")); err == nil {
					return defaultPath, nil
				}
			}
		}

		// Fallback: glob for any profile with cookies.sqlite
		for _, pattern := range []string{"*.default-release", "*.default*", "*"} {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern, "cookies.sqlite"))
			if len(matches) > 0 {
				return filepath.Dir(matches[0]), nil
			}
		}
	}

	return "", fmt.Errorf("no Firefox or Zen Browser profile found")
}

// parseDefaultProfile extracts the default profile path from profiles.ini.
func parseDefaultProfile(ini string, firefoxDir string) string {
	var currentPath string
	var currentIsRelative bool
	var isDefault bool

	for _, line := range strings.Split(ini, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			// New section — check if previous was default
			if isDefault && currentPath != "" {
				if currentIsRelative {
					return filepath.Join(firefoxDir, currentPath)
				}
				return currentPath
			}
			currentPath = ""
			currentIsRelative = false
			isDefault = false
			continue
		}
		if strings.HasPrefix(line, "Path=") {
			currentPath = strings.TrimPrefix(line, "Path=")
		}
		if line == "IsRelative=1" {
			currentIsRelative = true
		}
		if line == "Default=1" {
			isDefault = true
		}
	}
	// Check last section
	if isDefault && currentPath != "" {
		if currentIsRelative {
			return filepath.Join(firefoxDir, currentPath)
		}
		return currentPath
	}
	return ""
}
