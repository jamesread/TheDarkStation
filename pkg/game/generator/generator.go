package generator

import (
	"darkstation/pkg/engine/world"
)

// GridGenerator is an interface for map generation algorithms
type GridGenerator interface {
	Generate(level int) *world.Grid
	Name() string
}

// Available generators
var (
	LineWalker = &LineWalkerGenerator{}
	BSP        = &BSPGenerator{}
)

// DefaultGenerator is the default map generator
var DefaultGenerator GridGenerator = BSP
