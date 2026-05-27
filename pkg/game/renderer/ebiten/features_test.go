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

func TestEnvPlaquesEnabled_cvar(t *testing.T) {
	e := &EbitenRenderer{}
	setCvar("draw.env_plaques", "0")
	if e.EnvPlaquesEnabled() {
		t.Fatal("expected off with draw.env_plaques=0")
	}
	setCvar("draw.env_plaques", "1")
	if !e.EnvPlaquesEnabled() {
		t.Fatal("expected on with draw.env_plaques=1")
	}
	setCvar("draw.env_plaques", "0")
}
