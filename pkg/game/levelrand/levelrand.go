// Package levelrand provides a deterministic RNG for level generation and setup.
// All procedural placement must use this package (not math/rand global state).
package levelrand

import "math/rand"

var rng *rand.Rand

// Seed resets the level-generation RNG. Seed 0 is treated as 1.
func Seed(seed int64) {
	if seed == 0 {
		seed = 1
	}
	rng = rand.New(rand.NewSource(seed))
}

// RNG returns the active level-generation random source.
func RNG() *rand.Rand {
	if rng == nil {
		Seed(1)
	}
	return rng
}

// Intn returns a uniform int in [0,n).
func Intn(n int) int {
	return RNG().Intn(n)
}

// Float64 returns a uniform float64 in [0,1).
func Float64() float64 {
	return RNG().Float64()
}

// Float32 returns a uniform float32 in [0,1).
func Float32() float32 {
	return RNG().Float32()
}

// Shuffle shuffles items using the level RNG.
func Shuffle(n int, swap func(i, j int)) {
	RNG().Shuffle(n, swap)
}

// NewDerived returns an independent RNG stream from the level seed and tag (subsystems).
func NewDerived(levelSeed int64, tag uint64) *rand.Rand {
	if levelSeed == 0 {
		levelSeed = 1
	}
	return rand.New(rand.NewSource(levelSeed ^ int64(tag*0x9e3779b97f4a7c15)))
}
