// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// getTileFontSize returns the font size for map tiles, scaled to the current tile size
func (e *EbitenRenderer) getTileFontSize() float64 {
	// Scale font size based on tile size (default tile size is 24)
	return baseFontSize * float64(e.tileSize) / 24.0
}

// getUIFontSize returns the font size for UI text (50% of tile size)
func (e *EbitenRenderer) getUIFontSize() float64 {
	size := e.getTileFontSize() * 0.5
	if size < 10 {
		size = 10
	}
	return size
}

// getMonoFontFace returns a cached monospace font face for map tiles
func (e *EbitenRenderer) getMonoFontFace() *text.GoTextFace {
	size := e.getTileFontSize()
	if e.cachedMonoFace == nil || e.cachedTileFontSize != size {
		e.cachedTileFontSize = size
		e.cachedMonoFace = &text.GoTextFace{
			Source: e.monoFontSource,
			Size:   size,
		}
	}
	return e.cachedMonoFace
}

// getSansFontFace returns a cached sans-serif font face for UI text
func (e *EbitenRenderer) getSansFontFace() *text.GoTextFace {
	size := e.getUIFontSize()
	if e.cachedSansFace == nil || e.cachedUIFontSize != size {
		e.cachedUIFontSize = size
		e.cachedSansFace = &text.GoTextFace{
			Source: e.sansFontSource,
			Size:   size,
		}
	}
	return e.cachedSansFace
}

// getSansBoldFontFace returns a cached sans-serif bold font face (same size as UI)
func (e *EbitenRenderer) getSansBoldFontFace() *text.GoTextFace {
	size := e.getUIFontSize()
	if e.cachedSansBoldFace == nil || e.cachedUIFontSize != size {
		e.cachedUIFontSize = size
		e.cachedSansBoldFace = &text.GoTextFace{
			Source: e.sansBoldFontSource,
			Size:   size,
		}
	}
	return e.cachedSansBoldFace
}

// getSansBoldTitleFontFace returns a cached sans-serif bold font face 2pt larger than UI for menu titles
func (e *EbitenRenderer) getSansBoldTitleFontFace() *text.GoTextFace {
	size := e.getUIFontSize() + 2
	if e.cachedSansBoldTitleFace == nil || e.cachedSansBoldTitleSize != size {
		e.cachedSansBoldTitleSize = size
		e.cachedSansBoldTitleFace = &text.GoTextFace{
			Source: e.sansBoldFontSource,
			Size:   size,
		}
	}
	return e.cachedSansBoldTitleFace
}

// getMonoUIFontFace returns a monospace font face with UI font size (for console)
func (e *EbitenRenderer) getMonoUIFontFace() *text.GoTextFace {
	size := e.getUIFontSize()
	if e.cachedMonoUIFace == nil || e.cachedMonoUIFontSize != size {
		e.cachedMonoUIFontSize = size
		e.cachedMonoUIFace = &text.GoTextFace{
			Source: e.monoFontSource,
			Size:   size,
		}
	}
	return e.cachedMonoUIFace
}

// invalidateFontCache clears cached font faces (call when tile size changes)
func (e *EbitenRenderer) invalidateFontCache() {
	e.cachedMonoFace = nil
	e.cachedSansFace = nil
	e.cachedSansBoldFace = nil
	e.cachedSansBoldTitleFace = nil
	e.cachedMonoUIFace = nil
}
