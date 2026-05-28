package renderer

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// Power bar headroom thresholds (fraction of supply still available).
const (
	PowerBarHeadroomGreen  = 0.25
	PowerBarHeadroomOrange = 0.10
)

var (
	PowerBarColorGreen  = color.RGBA{60, 200, 100, 255}
	PowerBarColorOrange = color.RGBA{255, 165, 0, 255}
	PowerBarColorRed    = color.RGBA{255, 90, 90, 255}
	PowerBarColorTrack  = color.RGBA{35, 35, 48, 240}
	PowerBarColorBorder = color.RGBA{120, 120, 145, 255}
)

// PowerBarUsageColor returns the fill color for consumed power based on remaining headroom.
func PowerBarUsageColor(supply, consumption int) color.RGBA {
	if supply <= 0 {
		if consumption > 0 {
			return PowerBarColorRed
		}
		return PowerBarColorTrack
	}
	avail := float64(supply-consumption) / float64(supply)
	switch {
	case avail >= PowerBarHeadroomGreen:
		return PowerBarColorGreen
	case avail >= PowerBarHeadroomOrange:
		return PowerBarColorOrange
	default:
		return PowerBarColorRed
	}
}

// PowerBarUsageFraction returns the width fraction [0,1] for consumption relative to supply.
func PowerBarUsageFraction(supply, consumption int) float64 {
	if supply <= 0 {
		if consumption > 0 {
			return 1
		}
		return 0
	}
	f := float64(consumption) / float64(supply)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// PowerBarHighlightFraction returns the width fraction [0,1] for a highlighted load segment.
func PowerBarHighlightFraction(supply, highlight int) float64 {
	return PowerBarUsageFraction(supply, highlight)
}

// FormatPowerBarLine encodes a power bar row for menu/callout renderers (label|supply|consumption).
func FormatPowerBarLine(label string, supply, consumption int) string {
	return fmt.Sprintf("POWERBAR{%s|%d|%d}", label, supply, consumption)
}

// FormatPowerBarLineWithHighlight encodes a power bar row with an orange highlighted load segment.
func FormatPowerBarLineWithHighlight(label string, supply, consumption, highlight int) string {
	return fmt.Sprintf("POWERBAR{%s|%d|%d|%d}", label, supply, consumption, highlight)
}

// ParsePowerBarLine decodes FormatPowerBarLine. ok is false when line is not a power bar row.
func ParsePowerBarLine(line string) (label string, supply, consumption, highlight int, ok bool) {
	if !strings.HasPrefix(line, "POWERBAR{") || !strings.HasSuffix(line, "}") {
		return "", 0, 0, 0, false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, "POWERBAR{"), "}")
	parts := strings.Split(inner, "|")
	if len(parts) != 3 && len(parts) != 4 {
		return "", 0, 0, 0, false
	}
	supply, err1 := strconv.Atoi(parts[1])
	consumption, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil {
		return "", 0, 0, 0, false
	}
	if len(parts) == 4 {
		parsedHighlight, err := strconv.Atoi(parts[3])
		if err != nil {
			return "", 0, 0, 0, false
		}
		highlight = parsedHighlight
	}
	return parts[0], supply, consumption, highlight, true
}

// IsPowerBarLine reports whether s is a POWERBAR{…} encoded row.
func IsPowerBarLine(s string) bool {
	_, _, _, _, ok := ParsePowerBarLine(s)
	return ok
}

// FormatPowerBarWattsSuffix returns a compact "used/supply w" label for beside the bar.
func FormatPowerBarWattsSuffix(supply, consumption int) string {
	return fmt.Sprintf("%d/%dw", consumption, supply)
}
