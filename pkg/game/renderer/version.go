package renderer

import (
	"strings"
	"time"
)

// BuildLabel is the user-facing build stamp (friendly local date/time, minute precision).
var BuildLabel = "unknown"

// FormatBuildLabel converts a raw build timestamp (RFC3339 from release tooling, etc.)
// into a friendly label like "28 May 2026, 14:35". When raw is empty or unknown,
// now is used (typical for local go run / go build without ldflags).
func FormatBuildLabel(raw string, now time.Time) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "unknown" {
		return now.Local().Format("2 Jan 2006, 15:04")
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z0700",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Local().Format("2 Jan 2006, 15:04")
		}
	}
	return raw
}
