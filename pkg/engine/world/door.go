package world

// Door represents a door that connects a room to a corridor
// Doors can be unlocked by various conditions, starting with keycards
type Door struct {
	RoomName string // Name of the room this door belongs to
	Locked   bool
}

// NewDoor creates a new locked door for the given room
func NewDoor(roomName string) *Door {
	return &Door{
		RoomName: roomName,
		Locked:   true,
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
