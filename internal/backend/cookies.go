package backend

import (
	"encoding/json"
	"fmt"
	"strings"
)

var requiredCookies = []string{"SID", "HSID", "SSID", "APISID", "SAPISID", "OSID"}

// ParseCookies parses cookie input in JSON or semicolon-separated format.
// Returns a map of cookie name → value.
// Accepts:
//   - JSON: {"SID":"abc","HSID":"def",...}
//   - Semicolon-separated: SID=abc; HSID=def; ...
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
	} else {
		// Semicolon-separated: SID=abc; HSID=def
		for _, part := range strings.Split(input, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			eq := strings.IndexByte(part, '=')
			if eq < 0 {
				continue
			}
			key := strings.TrimSpace(part[:eq])
			val := strings.TrimSpace(part[eq+1:])
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
