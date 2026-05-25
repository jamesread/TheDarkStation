package features

import (
	"testing"

	engworld "darkstation/pkg/engine/world"
)

func TestVisitedSystemEnabled_defaultOffWithoutRenderer(t *testing.T) {
	if VisitedSystemEnabled() {
		t.Fatal("visited system should default off without renderer")
	}
}

func TestMarkVisited_noOpWhenDisabled(t *testing.T) {
	cell := &engworld.Cell{Visited: false}
	MarkVisited(cell)
	if cell.Visited {
		t.Fatal("MarkVisited should not set Visited when system disabled")
	}
}

func TestIsVisited_falseWhenDisabled(t *testing.T) {
	cell := &engworld.Cell{Visited: true}
	if IsVisited(cell) {
		t.Fatal("IsVisited should be false when system disabled even if cell.Visited set")
	}
}
