package renderer

import (
	"strings"
	"testing"
	"time"
)

func TestFormatBuildLabel_unknownUsesNow(t *testing.T) {
	now := time.Date(2026, 5, 28, 14, 35, 0, 0, time.Local)
	got := FormatBuildLabel("unknown", now)
	want := now.Local().Format("2 Jan 2006, 15:04")
	if got != want {
		t.Fatalf("FormatBuildLabel(unknown) = %q, want %q", got, want)
	}
}

func TestFormatBuildLabel_parsesRFC3339(t *testing.T) {
	got := FormatBuildLabel("2026-05-28T13:05:00Z", time.Now())
	if !strings.HasPrefix(got, "28 May 2026") || !strings.Contains(got, ":05") {
		t.Fatalf("FormatBuildLabel(RFC3339) = %q, want local 28 May 2026 with :05 minutes", got)
	}
}

func TestSetVersion_setsBuildLabel(t *testing.T) {
	SetVersion("1.2.3", "abc123", "2026-05-28T13:05:00Z")
	if BuildLabel == "" || BuildLabel == "dev" {
		t.Fatalf("BuildLabel = %q, want formatted build date", BuildLabel)
	}
}
