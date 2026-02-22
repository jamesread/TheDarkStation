package entities

import "testing"

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator("Gen-A", 3)
	if gen.Name != "Gen-A" {
		t.Errorf("Name = %q, want %q", gen.Name, "Gen-A")
	}
	if gen.BatteriesRequired != 3 {
		t.Errorf("BatteriesRequired = %d, want 3", gen.BatteriesRequired)
	}
	if gen.BatteriesInserted != 0 {
		t.Errorf("BatteriesInserted = %d, want 0", gen.BatteriesInserted)
	}
}

func TestGenerator_IsPowered(t *testing.T) {
	tests := []struct {
		name     string
		required int
		inserted int
		want     bool
	}{
		{"zero of one", 1, 0, false},
		{"one of two", 2, 1, false},
		{"exact match", 2, 2, true},
		{"over-inserted", 2, 3, true},
		{"zero required zero inserted", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := &Generator{BatteriesRequired: tt.required, BatteriesInserted: tt.inserted}
			if got := gen.IsPowered(); got != tt.want {
				t.Errorf("IsPowered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerator_BatteriesNeeded(t *testing.T) {
	tests := []struct {
		name     string
		required int
		inserted int
		want     int
	}{
		{"none inserted", 3, 0, 3},
		{"partial", 3, 1, 2},
		{"exact", 3, 3, 0},
		{"over-inserted", 3, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := &Generator{BatteriesRequired: tt.required, BatteriesInserted: tt.inserted}
			if got := gen.BatteriesNeeded(); got != tt.want {
				t.Errorf("BatteriesNeeded() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGenerator_InsertBatteries(t *testing.T) {
	tests := []struct {
		name           string
		required       int
		alreadyIn      int
		insert         int
		wantInserted   int
		wantTotal      int
		wantPowered    bool
	}{
		{"insert exact needed", 3, 0, 3, 3, 3, true},
		{"insert partial", 3, 0, 1, 1, 1, false},
		{"insert more than needed", 3, 0, 5, 3, 3, true},
		{"insert when already full", 3, 3, 2, 0, 3, true},
		{"insert zero", 3, 1, 0, 0, 1, false},
		{"top off remaining", 3, 2, 5, 1, 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := &Generator{BatteriesRequired: tt.required, BatteriesInserted: tt.alreadyIn}
			got := gen.InsertBatteries(tt.insert)
			if got != tt.wantInserted {
				t.Errorf("InsertBatteries(%d) returned %d, want %d", tt.insert, got, tt.wantInserted)
			}
			if gen.BatteriesInserted != tt.wantTotal {
				t.Errorf("BatteriesInserted = %d, want %d", gen.BatteriesInserted, tt.wantTotal)
			}
			if gen.IsPowered() != tt.wantPowered {
				t.Errorf("IsPowered() = %v, want %v", gen.IsPowered(), tt.wantPowered)
			}
		})
	}
}
