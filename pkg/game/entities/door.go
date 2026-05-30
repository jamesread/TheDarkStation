// Package entities contains game-specific entity types for The Dark Station.
// These are space station themed objects that extend the generic engine primitives.
package entities

// Door represents a door that connects a room to a corridor.
// Doors can be unlocked with keycards matching the room name.
type Door struct {
	RoomName     string // Name of the room this door belongs to
	Locked       bool
	KeycardGated bool // true for level keycard doors; stays passable without power once unlocked
}

// NewDoor creates a new locked keycard door for the given room.
func NewDoor(roomName string) *Door {
	return &Door{
		RoomName:     roomName,
		Locked:       true,
		KeycardGated: true,
	}
}

// NewUnlockedDoor creates a standard powered door (no keycard requirement).
func NewUnlockedDoor(roomName string) *Door {
	return &Door{
		RoomName:     roomName,
		Locked:       false,
		KeycardGated: false,
	}
}

// Unlock unlocks the door
func (d *Door) Unlock() {
	d.Locked = false
}

// KeycardName returns the keycard name required to unlock this door
func (d *Door) KeycardName() string {
	return d.RoomName + " Keycard"
}

// DoorName returns the display name for this door
func (d *Door) DoorName() string {
	return d.RoomName + " Door"
}
