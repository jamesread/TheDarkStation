// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"image/color"
	"math"
	"time"
)

// getPulsingExitColor returns a pulsing color for the unlocked exit icon
// Uses a sine wave to create a smooth pulsing effect
func (e *EbitenRenderer) getPulsingExitColor() color.Color {
	// Pulse period: 2 seconds (2000ms)
	const pulsePeriod = 2000.0
	now := time.Now().UnixMilli()

	// Calculate pulse value (0.0 to 1.0) using sine wave
	// This creates a smooth oscillation
	pulsePhase := float64(now%int64(pulsePeriod)) / pulsePeriod
	pulseValue := (math.Sin(pulsePhase*2*math.Pi) + 1.0) / 2.0 // 0.0 to 1.0

	// Pulse between 50% and 100% brightness
	minBrightness := 0.5
	maxBrightness := 1.0
	brightness := minBrightness + (maxBrightness-minBrightness)*pulseValue

	// Apply brightness to the base exit unlocked color (bright green)
	baseR, baseG, baseB, baseA := colorExitUnlocked.RGBA()
	r8 := uint8(float64(baseR>>8) * brightness)
	g8 := uint8(float64(baseG>>8) * brightness)
	b8 := uint8(float64(baseB>>8) * brightness)
	a8 := uint8(baseA >> 8)

	return color.RGBA{r8, g8, b8, a8}
}

// getPulsingExitBackgroundColor returns a pulsing background color for the unlocked exit
// Uses a distinct color (cyan/blue) that pulses
func (e *EbitenRenderer) getPulsingExitBackgroundColor() color.Color {
	// Pulse period: 2 seconds (2000ms)
	const pulsePeriod = 2000.0
	now := time.Now().UnixMilli()

	// Calculate pulse value (0.0 to 1.0) using sine wave
	pulsePhase := float64(now%int64(pulsePeriod)) / pulsePeriod
	pulseValue := (math.Sin(pulsePhase*2*math.Pi) + 1.0) / 2.0 // 0.0 to 1.0

	// Pulse between 30% and 70% brightness for background (distinct from icon)
	minBrightness := 0.3
	maxBrightness := 0.7
	brightness := minBrightness + (maxBrightness-minBrightness)*pulseValue

	// Use a distinct cyan/blue color for the background
	baseColor := color.RGBA{50, 255, 100, 255} // Greenish-cyan
	r8 := uint8(float64(baseColor.R) * brightness)
	g8 := uint8(float64(baseColor.G) * brightness)
	b8 := uint8(float64(baseColor.B) * brightness)

	return color.RGBA{r8, g8, b8, baseColor.A}
}
