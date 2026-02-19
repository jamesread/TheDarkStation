// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"image/color"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"darkstation/pkg/game/renderer"
	"github.com/leonelquinteros/gotext"
)

// dynamicGet is used for runtime translation key lookups.
// We use a function variable to avoid go vet's non-constant format string check,
// since we intentionally look up translation keys dynamically from markup.
var dynamicGet = gotext.Get

// textSegment represents a segment of text with a specific color
type textSegment struct {
	text  string
	color color.Color
}

// drawColoredChar draws a character with color at the given tile position (uses mono font)
func (e *EbitenRenderer) drawColoredChar(screen *ebiten.Image, char string, x, y int, col color.Color) {
	e.drawColoredCharF(screen, char, float64(x), float64(y), col)
}

// drawColoredCharF is the float64 variant for sub-pixel positioning (smooth camera).
func (e *EbitenRenderer) drawColoredCharF(screen *ebiten.Image, char string, x, y float64, col color.Color) {
	face := e.getMonoFontFace()

	// Calculate position to center the character in the tile
	// text.Measure returns the bounding box width and height
	w, h := text.Measure(char, face, 0)

	// Center horizontally and vertically within the tile
	// text/v2 Draw uses top-left as the origin point
	offsetX := (float64(e.tileSize) - w) / 2
	offsetY := (float64(e.tileSize) - h) / 2

	op := &text.DrawOptions{}
	op.GeoM.Translate(x+offsetX, y+offsetY)
	op.ColorScale.ScaleWithColor(col)

	text.Draw(screen, char, face, op)
}

// drawColoredText draws text with a specific color using sans-serif font for UI
// Translates the string using gettext before drawing.
// If the string is not a translation key, gotext.Get will return it unchanged.
func (e *EbitenRenderer) drawColoredText(screen *ebiten.Image, str string, x, y int, col color.Color) {
	e.drawColoredTextWithFace(screen, str, x, y, col, e.getSansFontFace())
}

// drawColoredTextWithFace draws text with a specific color and font face.
// Translates the string using gettext before drawing.
// Uses the face's size for baseline offset so different font sizes position correctly.
func (e *EbitenRenderer) drawColoredTextWithFace(screen *ebiten.Image, str string, x, y int, col color.Color, face *text.GoTextFace) {
	translated := gotext.Get(str)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y)+face.Size)
	op.ColorScale.ScaleWithColor(col)

	text.Draw(screen, translated, face, op)
}

// hasTitleMarkup returns true if the string contains title-style markup (TITLE{}, UNPOWERED{}, UNPOWERED_SUBTLE{}).
// Used to apply title face (bold, larger) to the first line of callouts.
func hasTitleMarkup(s string) bool {
	return strings.Contains(s, "TITLE{") || strings.Contains(s, "UNPOWERED{") || strings.Contains(s, "UNPOWERED_SUBTLE{")
}

// getTitleColorFromLine returns the accent color from the first marked-up segment in the line, or colorAction as fallback.
// Used to derive tooltip border color from the title's markup (e.g. UNPOWERED{} -> red, TITLE{} -> colorAction).
func (e *EbitenRenderer) getTitleColorFromLine(line string) color.Color {
	segments := e.parseMarkup(line)
	for _, seg := range segments {
		if seg.color != colorText {
			return seg.color
		}
	}
	return colorAction
}

// parseMarkup parses a message string with markup (ITEM{}, ROOM{}, ACTION{}, GT{}) and returns colored segments
func (e *EbitenRenderer) parseMarkup(msg string) []textSegment {
	var segments []textSegment
	// Regex to match markup: FUNCTION{content} (FUNCTION can include underscores, e.g. UNPOWERED_SUBTLE)
	markupRegex := regexp.MustCompile(`([A-Z][A-Z0-9_]*)\{([^}]*)\}`)

	lastIndex := 0
	matches := markupRegex.FindAllStringSubmatchIndex(msg, -1)

	for _, match := range matches {
		// Add text before the markup
		if match[0] > lastIndex {
			plainText := msg[lastIndex:match[0]]
			if plainText != "" {
				segments = append(segments, textSegment{text: plainText, color: colorText})
			}
		}

		// Extract function name and content
		function := msg[match[2]:match[3]]
		content := msg[match[4]:match[5]]

		var segColor color.Color
		switch function {
		case "ITEM":
			segColor = colorItem
		case "ROOM":
			segColor = colorFloorVisited // Light gray-blue for room names
		case "ACTION":
			segColor = colorAction
		case "POWERED":
			segColor = colorExitUnlocked // Bright green for exit/lift
		case "UNPOWERED_SUBTLE":
			segColor = colorUnpoweredSubtle // Muted gray when unpowered due to room terminal dependency (before UNPOWERED)
		case "UNPOWERED":
			segColor = colorHazard // Red for unpowered state
		case "GT":
			// GT{} is for translations - look up the translation
			content = dynamicGet(content)
			segColor = colorText
		case "FURNITURE":
			// FURNITURE{} uses the furniture callout color (tan/brown for checked furniture)
			segColor = renderer.CalloutColorFurnitureChecked
		case "HAZARD":
			// HAZARD{} uses the hazard color (red)
			segColor = colorHazard
		case "SUBTLE":
			// SUBTLE{} uses the same color as labels (e.g. "Power supply:")
			segColor = colorSubtle
		case "LOCATION":
			// LOCATION{} uses a soft blue-gray for room names and location labels
			segColor = colorLocation
		case "DOOR":
			// DOOR{} uses the door/locked color (yellow for locked doors)
			segColor = renderer.CalloutColorDoor
		case "TITLE":
			// TITLE{} uses the standard title color (blue-purple, matches menu titles)
			segColor = colorAction
		default:
			segColor = colorText
		}

		segments = append(segments, textSegment{text: content, color: segColor})
		lastIndex = match[1]
	}

	// Add remaining text after last markup
	if lastIndex < len(msg) {
		plainText := msg[lastIndex:]
		if plainText != "" {
			segments = append(segments, textSegment{text: plainText, color: colorText})
		}
	}

	// If no markup found, return the whole message as a single segment
	if len(segments) == 0 {
		segments = append(segments, textSegment{text: msg, color: colorText})
	}

	return segments
}

// applyAlpha applies an alpha value to a color
func (e *EbitenRenderer) applyAlpha(c color.Color, alpha float64) color.Color {
	if alpha <= 0 {
		alpha = 0
	}
	if alpha > 1.0 {
		alpha = 1.0
	}

	r, g, b, a := c.RGBA()
	// RGBA returns values in 0-65535 range, convert to 0-255
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)
	a8 := uint8(a >> 8)

	// Apply alpha to both RGB and alpha channel for proper fade from black
	// This ensures colors fade to transparent black, not transparent bright colors
	newR := uint8(float64(r8) * alpha)
	newG := uint8(float64(g8) * alpha)
	newB := uint8(float64(b8) * alpha)
	newAlpha := uint8(float64(a8) * alpha)

	return color.RGBA{newR, newG, newB, newAlpha}
}

// drawColoredTextSegments draws multiple text segments with different colors
func (e *EbitenRenderer) drawColoredTextSegments(screen *ebiten.Image, segments []textSegment, x, y int) {
	e.drawColoredTextSegmentsWithFace(screen, segments, x, y, e.getSansFontFace())
}

// drawColoredTextSegmentsWithFace draws multiple text segments with a specific font face (e.g. title font for first line of callouts).
func (e *EbitenRenderer) drawColoredTextSegmentsWithFace(screen *ebiten.Image, segments []textSegment, x, y int, face *text.GoTextFace) {
	currentX := float64(x)

	for _, seg := range segments {
		if seg.text == "" {
			continue
		}

		op := &text.DrawOptions{}
		op.GeoM.Translate(currentX, float64(y)+face.Size)
		op.ColorScale.ScaleWithColor(seg.color)

		text.Draw(screen, seg.text, face, op)

		w, _ := text.Measure(seg.text, face, 0)
		currentX += w
	}
}

// getTextWidth returns the width of a string in pixels at UI font size
func (e *EbitenRenderer) getTextWidth(str string) float64 {
	face := e.getSansFontFace()
	w, _ := text.Measure(str, face, 0)
	return w
}

// getTextWidthWithFace returns the width of a string in pixels using the given font face.
func (e *EbitenRenderer) getTextWidthWithFace(str string, face *text.GoTextFace) float64 {
	w, _ := text.Measure(str, face, 0)
	return w
}
