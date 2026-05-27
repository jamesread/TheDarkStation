package state

import "testing"

func TestFormatRunDuration(t *testing.T) {
	if got := FormatRunDuration(45); got == "" {
		t.Fatal("expected non-empty duration string")
	}
	if got := FormatRunDuration(3661); got == "" {
		t.Fatal("expected non-empty duration string for long runs")
	}
}
