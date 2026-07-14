package renderer

import (
	"fmt"
	"strings"
)

const (
	ItemIconKey     = "K"
	ItemIconMap     = "M"
	ItemIconGeneric = "?"
	ItemIconBattery = "■"
)

// InventoryDepictionKey identifies the map-style colors for an inventory row glyph.
type InventoryDepictionKey string

const (
	InventoryDepictionKeycard InventoryDepictionKey = "keycard"
	InventoryDepictionKeyBattery InventoryDepictionKey = "battery"
	InventoryDepictionKeyMap    InventoryDepictionKey = "map"
	InventoryDepictionKeyItem   InventoryDepictionKey = "item"
)

// InventoryDepiction describes how an inventory entry appears on the grid.
type InventoryDepiction struct {
	Icon string
	Key  InventoryDepictionKey
}

// InventoryDepictionForName returns the cell glyph used for a carried item name.
func InventoryDepictionForName(name string) InventoryDepiction {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "keycard"):
		return InventoryDepiction{Icon: ItemIconKey, Key: InventoryDepictionKeycard}
	case strings.Contains(lower, "battery"):
		return InventoryDepiction{Icon: ItemIconBattery, Key: InventoryDepictionKeyBattery}
	case name == "Map":
		return InventoryDepiction{Icon: ItemIconMap, Key: InventoryDepictionKeyMap}
	default:
		return InventoryDepiction{Icon: ItemIconGeneric, Key: InventoryDepictionKeyItem}
	}
}

// InventoryDepictionForMap returns the depiction for the run-wide map item.
func InventoryDepictionForMap() InventoryDepiction {
	return InventoryDepiction{Icon: ItemIconMap, Key: InventoryDepictionKeyMap}
}

// InventoryDepictionForBatteries returns the depiction for battery stacks.
func InventoryDepictionForBatteries() InventoryDepiction {
	return InventoryDepiction{Icon: ItemIconBattery, Key: InventoryDepictionKeyBattery}
}

// FormatInventoryRowLine encodes an inventory menu row with icon + label.
func FormatInventoryRowLine(dep InventoryDepiction, label string) string {
	return fmt.Sprintf("INVROW{%s|%s|%s}", dep.Icon, dep.Key, label)
}

// ParseInventoryRowLine decodes an INVROW{icon|key|label} menu row.
func ParseInventoryRowLine(line string) (icon, key, label string, ok bool) {
	if !IsInventoryRowLine(line) {
		return "", "", "", false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, "INVROW{"), "}")
	parts := strings.SplitN(inner, "|", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// IsInventoryRowLine reports whether s is an INVROW{…} encoded row.
func IsInventoryRowLine(s string) bool {
	return strings.HasPrefix(s, "INVROW{") && strings.HasSuffix(s, "}")
}
