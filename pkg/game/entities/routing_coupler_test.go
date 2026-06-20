package entities

import "testing"

func TestRoutingCouplerDifficulty_scalesWithDeck(t *testing.T) {
	easy := RoutingCouplerDifficulty(3)
	hard := RoutingCouplerDifficulty(10)
	if easy.Axes >= hard.Axes && easy.LockThreshold <= hard.LockThreshold {
		t.Fatalf("expected harder decks to need more axes and/or lower threshold: easy=%+v hard=%+v", easy, hard)
	}
	if easy.Step <= hard.Step {
		t.Fatalf("expected smaller step (finer control) on harder decks: easy=%d hard=%d", easy.Step, hard.Step)
	}
	if easy.MaxDist <= hard.MaxDist {
		t.Fatalf("expected tighter tolerance on harder decks: easy=%d hard=%d", easy.MaxDist, hard.MaxDist)
	}
}

func TestRoutingCouplerSignalLock_perfectAlignment(t *testing.T) {
	targets := []int{40, 60}
	values := []int{40, 60}
	lock := RoutingCouplerSignalLock(targets, values, 30)
	if lock < 0.999 {
		t.Fatalf("perfect alignment lock = %v, want ~1", lock)
	}
}

func TestRoutingCouplerLocked_requiresThreshold(t *testing.T) {
	params := RoutingCouplerDifficulty(8)
	targets := RoutingCouplerTargets(42, params.Axes)
	values := make([]int, len(targets))
	copy(values, targets)
	if !RoutingCouplerLocked(targets, values, params) {
		t.Fatal("exact targets should satisfy lock threshold")
	}
	for i := range values {
		values[i] = (values[i] + 50) % 101
	}
	if RoutingCouplerLocked(targets, values, params) {
		t.Fatal("far misalignment should not satisfy lock threshold")
	}
}

func TestRoutingTargetLevel_prefersTargetDeckID(t *testing.T) {
	repair := NewRepairObjective("routing-repair-deck9-0", RepairSignalCalibrator, "Lab", 0, 0)
	repair.Name = "Lift Routing Coupler (Deck 9)"
	repair.TargetDeckID = 4
	if got := repair.RoutingTargetLevel(); got != 5 {
		t.Fatalf("RoutingTargetLevel() = %d, want 5 from TargetDeckID", got)
	}
}

func TestRoutingTargetLevel_parsesNameFallback(t *testing.T) {
	repair := NewRepairObjective("routing-repair-deck7-fallback", RepairSignalCalibrator, "Lab", 0, 0)
	repair.Name = "Lift Routing Coupler (Deck 7)"
	if got := repair.RoutingTargetLevel(); got != 7 {
		t.Fatalf("RoutingTargetLevel() = %d, want 7 from name", got)
	}
}

func TestRoutingCouplerTargets_deterministic(t *testing.T) {
	a := RoutingCouplerTargets(99, 3)
	b := RoutingCouplerTargets(99, 3)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("targets differ at %d: %v vs %v", i, a, b)
		}
	}
}
