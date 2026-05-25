package ebiten

import "testing"

func TestVisitedSystemEnabled_cvar(t *testing.T) {
	e := &EbitenRenderer{}
	setCvar("gameplay.visited", "0")
	if e.VisitedSystemEnabled() {
		t.Fatal("expected off with gameplay.visited=0")
	}
	setCvar("gameplay.visited", "1")
	if !e.VisitedSystemEnabled() {
		t.Fatal("expected on with gameplay.visited=1")
	}
	setCvar("gameplay.visited", "0")
}
