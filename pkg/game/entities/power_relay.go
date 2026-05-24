package entities

// PowerRelay is a corridor routing switch (Story 6 / power-routing Phase 3).
// When Closed is true, control power may propagate through the cell; when false, the path is open.
type PowerRelay struct {
	Closed bool
}

// NewPowerRelay returns a relay defaulting to closed (conducting).
func NewPowerRelay() *PowerRelay {
	return &PowerRelay{Closed: true}
}

// NewPowerRelayOpen returns a relay that blocks mesh propagation until the player closes it.
func NewPowerRelayOpen() *PowerRelay {
	return &PowerRelay{Closed: false}
}
