package renderer

import "testing"

func TestPowerBarUsageColor_headroomBands(t *testing.T) {
	green := PowerBarUsageColor(100, 70)  // 30% free
	orange := PowerBarUsageColor(100, 80) // 20% free
	red := PowerBarUsageColor(100, 92)    // 8% free
	if green != PowerBarColorGreen {
		t.Fatalf("30%% headroom: got %+v want green", green)
	}
	if orange != PowerBarColorOrange {
		t.Fatalf("20%% headroom: got %+v want orange", orange)
	}
	if red != PowerBarColorRed {
		t.Fatalf("8%% headroom: got %+v want red", red)
	}
}

func TestPowerBarUsageColor_zeroSupplyWithLoad(t *testing.T) {
	if PowerBarUsageColor(0, 10) != PowerBarColorRed {
		t.Fatal("expected red when supply is 0 but load > 0")
	}
}

func TestParsePowerBarLine_roundTrip(t *testing.T) {
	line := FormatPowerBarLine("Grid power", 100, 75)
	label, supply, consumption, highlight, ok := ParsePowerBarLine(line)
	if !ok || label != "Grid power" || supply != 100 || consumption != 75 || highlight != 0 {
		t.Fatalf("ParsePowerBarLine(%q) = %q %d %d %d %v", line, label, supply, consumption, highlight, ok)
	}
}

func TestParsePowerBarLine_roundTripWithHighlight(t *testing.T) {
	line := FormatPowerBarLineWithHighlight("Grid power", 100, 75, 15)
	label, supply, consumption, highlight, ok := ParsePowerBarLine(line)
	if !ok || label != "Grid power" || supply != 100 || consumption != 75 || highlight != 15 {
		t.Fatalf("ParsePowerBarLine(%q) = %q %d %d %d %v", line, label, supply, consumption, highlight, ok)
	}
}

func TestPowerBarUsageFraction_clampsOverload(t *testing.T) {
	if f := PowerBarUsageFraction(100, 150); f != 1 {
		t.Fatalf("overload fraction = %v, want 1", f)
	}
}

func TestPowerBarHighlightFraction_usesSupply(t *testing.T) {
	if f := PowerBarHighlightFraction(100, 25); f != 0.25 {
		t.Fatalf("highlight fraction = %v, want 0.25", f)
	}
}

func TestPowerBarUsageColor_exactThresholds(t *testing.T) {
	at25 := PowerBarUsageColor(100, 75)
	if at25 != PowerBarColorGreen {
		t.Fatalf("exactly 25%% headroom should be green, got %+v", at25)
	}
	at10 := PowerBarUsageColor(100, 90)
	if at10 != PowerBarColorOrange {
		t.Fatalf("exactly 10%% headroom should be orange, got %+v", at10)
	}
}
