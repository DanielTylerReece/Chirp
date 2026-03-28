package backend

import (
	"encoding/json"
	"fmt"
	"strings"
)

var requiredCookies = []string{"SID", "HSID", "SSID", "APISID", "SAPISID"}

// ParseCookies parses cookie input in multiple formats.
// Returns a map of cookie name → value.
// Accepts:
//   - JSON: {"SID":"abc","HSID":"def",...}
//   - Semicolon-separated: SID=abc; HSID=def; ...
//   - Colon-quoted (one per line): SID:"abc"\nHSID:"def"
//   - Colon-equals (one per line): SID:abc or SID=abc
func ParseCookies(input string) (map[string]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty cookie input")
	}

	cookies := make(map[string]string)

	// Try JSON first
	if strings.HasPrefix(input, "{") {
		if err := json.Unmarshal([]byte(input), &cookies); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	} else if strings.Contains(input, "\n") {
		// Line-separated: KEY:"value" or KEY:value or KEY=value
		for _, line := range strings.Split(input, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Try colon separator first (KEY:"value" or KEY:value)
			sep := strings.IndexByte(line, ':')
			if sep < 0 {
				// Try equals
				sep = strings.IndexByte(line, '=')
			}
			if sep < 0 {
				continue
			}
			key := strings.TrimSpace(line[:sep])
			val := strings.TrimSpace(line[sep+1:])
			// Strip surrounding quotes
			val = strings.Trim(val, "\"'")
			if key != "" && val != "" {
				cookies[key] = val
			}
		}
	} else {
		// Semicolon-separated: SID=abc; HSID=def
		for _, part := range strings.Split(input, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			sep := strings.IndexByte(part, '=')
			if sep < 0 {
				sep = strings.IndexByte(part, ':')
			}
			if sep < 0 {
				continue
			}
			key := strings.TrimSpace(part[:sep])
			val := strings.TrimSpace(part[sep+1:])
			val = strings.Trim(val, "\"'")
			if key != "" && val != "" {
				cookies[key] = val
			}
		}
	}

	// Validate required cookies
	var missing []string
	for _, name := range requiredCookies {
		if cookies[name] == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required cookies: %s", strings.Join(missing, ", "))
	}

	return cookies, nil
}
