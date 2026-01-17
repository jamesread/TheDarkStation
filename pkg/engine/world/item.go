package world

import (
	"github.com/zyedidia/generic/mapset"
)

// ItemSet is a set of items
type ItemSet = mapset.Set[*Item]

// Item represents a collectible item in the world
type Item struct {
	Name string
}

// NewItem creates a new item with the given name
func NewItem(name string) *Item {
	return &Item{Name: name}
}
