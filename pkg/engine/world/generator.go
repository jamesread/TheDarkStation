package world

// Generator represents a power generator that requires batteries to activate
type Generator struct {
	Name              string
	BatteriesRequired int
	BatteriesInserted int
	Cell              *Cell
}

// NewGenerator creates a new unpowered generator
func NewGenerator(name string, batteriesRequired int) *Generator {
	return &Generator{
		Name:              name,
		BatteriesRequired: batteriesRequired,
		BatteriesInserted: 0,
	}
}

// IsPowered returns true if the generator has enough batteries
func (g *Generator) IsPowered() bool {
	return g.BatteriesInserted >= g.BatteriesRequired
}

// BatteriesNeeded returns how many more batteries are needed
func (g *Generator) BatteriesNeeded() int {
	needed := g.BatteriesRequired - g.BatteriesInserted
	if needed < 0 {
		return 0
	}
	return needed
}

// InsertBatteries adds batteries to the generator, returns how many were actually inserted
func (g *Generator) InsertBatteries(count int) int {
	needed := g.BatteriesNeeded()
	if count > needed {
		count = needed
	}
	g.BatteriesInserted += count
	return count
}
