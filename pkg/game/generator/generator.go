package generator

import (
	"darkcastle/pkg/engine/world"
)

// GridGenerator is an interface for map generation algorithms
type GridGenerator interface {
	Generate(level int) *world.Grid
	Name() string
}

// DefaultGenerator is the default map generator
var DefaultGenerator GridGenerator = &LineWalkerGenerator{}
