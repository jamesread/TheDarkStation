package renderer

import "testing"

func TestFormatPowerLoad_zeroDrawPowered(t *testing.T) {
	got := FormatPowerLoad(0, true, false)
	if got != "POWERED{0w}" {
		t.Fatalf("FormatPowerLoad(0, powered) = %q, want POWERED{0w}", got)
	}
}

func TestFormatPowerLoad_zeroDrawUnpowered(t *testing.T) {
	got := FormatPowerLoad(0, false, false)
	if got != "UNPOWERED{0w}" {
		t.Fatalf("FormatPowerLoad(0, unpowered) = %q, want UNPOWERED{0w}", got)
	}
}
