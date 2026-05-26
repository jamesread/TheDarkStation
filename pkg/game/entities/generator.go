package entities

// Generator represents a power generator that requires batteries to activate
type Generator struct {
	Name              string
	BatteriesRequired int
	BatteriesInserted int
	Online            bool // Running after startup sequence (hold USE)
	Tripped           bool // Overload shut down; batteries may remain until restart
}

// NewGenerator creates a new unpowered generator
func NewGenerator(name string, batteriesRequired int) *Generator {
	return &Generator{
		Name:              name,
		BatteriesRequired: batteriesRequired,
		BatteriesInserted: 0,
	}
}

// IsPowered returns true when online, not tripped, and enough batteries are installed.
func (g *Generator) IsPowered() bool {
	return g != nil && g.Online && !g.Tripped && g.BatteriesInserted >= g.BatteriesRequired
}

// HasEnoughBatteries reports whether the generator has enough batteries installed.
func (g *Generator) HasEnoughBatteries() bool {
	return g != nil && g.BatteriesInserted >= g.BatteriesRequired
}

// NeedsStartupSequence reports whether the generator is fueled but awaiting hold-to-use startup.
func (g *Generator) NeedsStartupSequence() bool {
	return g != nil && !g.IsPowered() && g.HasEnoughBatteries()
}

// Trip shuts the generator down after an overload (player must bring it back online).
func (g *Generator) Trip() {
	if g != nil {
		g.Tripped = true
		g.Online = false
	}
}

// BringOnline starts the generator when it has enough batteries and is not tripped.
func (g *Generator) BringOnline() bool {
	if g == nil || g.Tripped || !g.HasEnoughBatteries() {
		return false
	}
	g.Online = true
	return true
}

// Restart clears a trip and brings the generator back online when it has enough batteries.
func (g *Generator) Restart() bool {
	if g == nil || !g.HasEnoughBatteries() {
		return false
	}
	g.Tripped = false
	return g.BringOnline()
}

// BatteriesNeeded returns how many more batteries are needed
func (g *Generator) BatteriesNeeded() int {
	needed := g.BatteriesRequired - g.BatteriesInserted
	if needed < 0 {
		return 0
	}
	return needed
}

// InsertBatteries adds batteries to the generator, returns how many were actually inserted.
// Does not start the generator; call BringOnline after the startup sequence.
func (g *Generator) InsertBatteries(count int) int {
	needed := g.BatteriesNeeded()
	if count > needed {
		count = needed
	}
	g.BatteriesInserted += count
	return count
}

// InsertBatteriesAndStart inserts batteries and brings the generator online when fully fueled.
// Used by level spawn and tests that need an already-running generator.
func (g *Generator) InsertBatteriesAndStart(count int) int {
	inserted := g.InsertBatteries(count)
	if g.HasEnoughBatteries() {
		g.BringOnline()
	}
	return inserted
}
