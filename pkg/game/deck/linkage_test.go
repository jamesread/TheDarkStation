package deck

import "testing"

func TestMultiHopLinkageActive(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		minimal  bool
		expected bool
	}{
		{"below tier", 4, false, false},
		{"tier on", 5, false, true},
		{"final minimal deck gate", TotalDecks, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MultiHopLinkageActive(tt.level, tt.minimal); got != tt.expected {
				t.Fatalf("MultiHopLinkageActive(%d,%v) = %v want %v", tt.level, tt.minimal, got, tt.expected)
			}
		})
	}
}
