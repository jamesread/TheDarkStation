// Package levelseed formats and parses level generation seeds for display and debug entry.
package levelseed

import (
	"fmt"
	"strconv"
	"strings"
)

// Format returns the seed as uppercase hexadecimal (no 0x prefix).
func Format(seed int64) string {
	return strings.ToUpper(fmt.Sprintf("%X", uint64(seed)))
}

// Parse accepts uppercase or lowercase hex, with an optional 0x prefix.
func Parse(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.EqualFold(s[:2], "0x") {
		s = s[2:]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("seed is required")
	}
	u, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid hex seed: %v", err)
	}
	return int64(u), nil
}
