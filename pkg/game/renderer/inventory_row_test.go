package renderer

import "testing"

func TestInventoryDepictionForName(t *testing.T) {
	dep := InventoryDepictionForName("Reactor Keycard")
	if dep.Icon != ItemIconKey || dep.Key != InventoryDepictionKeycard {
		t.Fatalf("keycard dep = %+v", dep)
	}
	dep = InventoryDepictionForName("Spare Battery")
	if dep.Icon != ItemIconBattery || dep.Key != InventoryDepictionKeyBattery {
		t.Fatalf("battery dep = %+v", dep)
	}
}

func TestParseInventoryRowLine(t *testing.T) {
	line := FormatInventoryRowLine(InventoryDepictionForMap(), "ITEM{Map}")
	icon, key, label, ok := ParseInventoryRowLine(line)
	if !ok || icon != ItemIconMap || key != string(InventoryDepictionKeyMap) || label != "ITEM{Map}" {
		t.Fatalf("parse = %q %q %q ok=%v", icon, key, label, ok)
	}
}
