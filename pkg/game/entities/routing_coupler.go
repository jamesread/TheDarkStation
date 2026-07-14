package entities

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
)

const routingCouplerValueMax = 100

// RoutingCouplerParams describes alignment mini-game difficulty for a target deck.
type RoutingCouplerParams struct {
	Axes          int
	LockThreshold float64
	Step          int
	MaxDist       int
}

// RoutingCouplerDifficulty returns mini-game parameters scaled to the deck being unlocked.
func RoutingCouplerDifficulty(targetLevel int) RoutingCouplerParams {
	if targetLevel < 2 {
		targetLevel = 2
	}
	switch {
	case targetLevel <= 3:
		return RoutingCouplerParams{Axes: 1, LockThreshold: 0.92, Step: 6, MaxDist: 36}
	case targetLevel <= 5:
		return RoutingCouplerParams{Axes: 2, LockThreshold: 0.88, Step: 5, MaxDist: 30}
	case targetLevel <= 7:
		return RoutingCouplerParams{Axes: 2, LockThreshold: 0.85, Step: 4, MaxDist: 24}
	default:
		return RoutingCouplerParams{Axes: 3, LockThreshold: 0.82, Step: 3, MaxDist: 20}
	}
}

// RoutingCouplerAxisNames returns the labels for each alignment axis.
func RoutingCouplerAxisNames(axes int) []string {
	names := []string{"Phase", "Gain", "Bias"}
	if axes <= 0 {
		return nil
	}
	if axes > len(names) {
		axes = len(names)
	}
	return names[:axes]
}

// RoutingCouplerSeed derives a deterministic seed for one routing coupler repair.
func RoutingCouplerSeed(levelSeed int64, repairID string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(repairID))
	return levelSeed ^ int64(h.Sum64())
}

// RoutingCouplerTargets returns hidden alignment targets for each axis.
func RoutingCouplerTargets(seed int64, axes int) []int {
	if axes <= 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(seed ^ 0x243f6a8885a308d3))
	targets := make([]int, axes)
	for i := range targets {
		targets[i] = 15 + rng.Intn(71)
	}
	return targets
}

// RoutingCouplerInitialValues returns starting knob positions offset from the targets.
func RoutingCouplerInitialValues(seed int64, targets []int, targetLevel int) []int {
	if len(targets) == 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(int64(uint64(seed) ^ uint64(0x9e3779b97f4a7c15))))
	spread := 20 + targetLevel*2
	values := make([]int, len(targets))
	for i, target := range targets {
		offset := spread/2 + rng.Intn(spread)
		if rng.Intn(2) == 0 {
			offset = -offset
		}
		values[i] = clampRoutingCouplerValue(target + offset)
	}
	return values
}

// RoutingCouplerSignalLock returns 0..1 alignment quality (1 = perfect lock).
func RoutingCouplerSignalLock(targets, values []int, maxDist int) float64 {
	if len(targets) == 0 || len(targets) != len(values) || maxDist <= 0 {
		return 0
	}
	var sum float64
	for i := range targets {
		sum += routingCouplerAxisScore(targets[i], values[i], maxDist)
	}
	return sum / float64(len(targets))
}

// RoutingCouplerLocked reports whether alignment meets the lock threshold.
func RoutingCouplerLocked(targets, values []int, params RoutingCouplerParams) bool {
	return RoutingCouplerSignalLock(targets, values, params.MaxDist) >= params.LockThreshold
}

// RoutingCouplerAdjustValue nudges one axis by step, wrapping at the ends.
func RoutingCouplerAdjustValue(value, delta, step int) int {
	return clampRoutingCouplerValue(value + delta*step)
}

func routingCouplerAxisScore(target, value, maxDist int) float64 {
	dist := math.Abs(float64(value - target))
	if dist >= float64(maxDist) {
		return 0
	}
	return 1 - dist/float64(maxDist)
}

func clampRoutingCouplerValue(v int) int {
	if v < 0 {
		return 0
	}
	if v > routingCouplerValueMax {
		return routingCouplerValueMax
	}
	return v
}

// RoutingTargetLevel returns the 1-based deck level this routing coupler unlocks.
func (r *RepairObjective) RoutingTargetLevel() int {
	if r == nil {
		return 0
	}
	if r.TargetDeckID >= 0 {
		return r.TargetDeckID + 1
	}
	var level int
	if _, err := fmt.Sscanf(r.Name, "Lift Routing Coupler (Deck %d)", &level); err == nil && level > 0 {
		return level
	}
	return 0
}
