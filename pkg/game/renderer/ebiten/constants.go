// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import "image/color"

// Color palette for the game - brighter colors for visibility
var (
	colorBackground      = color.RGBA{26, 26, 46, 255}    // Dark blue-gray
	colorMapBackground   = color.RGBA{15, 15, 26, 255}    // Darker for map area
	colorPlayer          = color.RGBA{0, 255, 0, 255}     // Bright green
	colorWall            = color.RGBA{180, 180, 200, 255} // Light gray-blue for wall text
	colorWallBg          = color.RGBA{60, 60, 80, 255}    // Darker background for walls
	colorWallHighlight   = color.RGBA{90, 58, 18, 255}    // Dark orange tint matching maintenance terminal for selected room walls
	colorWallBgPowered   = color.RGBA{40, 80, 40, 255}    // Dark green background for walls in powered rooms
	colorFloor           = color.RGBA{100, 100, 120, 255} // Medium gray for undiscovered
	colorFloorVisited    = color.RGBA{160, 160, 180, 255} // Lighter gray for visited
	colorFloorBg         = color.RGBA{38, 38, 58, 255}    // Dark blue-gray background for floor cells
	colorFloorVisitedBg  = color.RGBA{44, 44, 64, 255}    // Slightly lighter floor background for visited
	colorDoorLocked      = color.RGBA{255, 255, 0, 255}   // Bright yellow
	colorDoorUnlocked    = color.RGBA{0, 220, 0, 255}     // Bright green
	colorKeycard         = color.RGBA{100, 150, 255, 255} // Bright blue
	colorItem            = color.RGBA{210, 185, 110, 255} // Tan / sand — floor pickups, ITEM{} tooltip markup
	colorBattery         = color.RGBA{255, 200, 100, 255} // Orange for batteries
	colorHazard          = color.RGBA{255, 80, 80, 255}   // Bright red
	colorHazardCtrl      = color.RGBA{255, 150, 200, 255} // Pink for circuit breakers
	colorGeneratorOff    = color.RGBA{255, 100, 100, 255} // Bright red
	colorGeneratorOn     = color.RGBA{0, 255, 100, 255}   // Bright green
	colorTerminal        = color.RGBA{100, 150, 255, 255} // Bright blue
	colorTerminalUsed    = color.RGBA{120, 120, 140, 255} // Medium gray
	colorMaintenance     = color.RGBA{255, 165, 0, 255}   // Orange for maintenance terminals
	colorMaintenanceBg   = color.RGBA{58, 38, 12, 255}    // Dark orange tile plate (pairs with colorMaintenance)
	colorFurniture       = color.RGBA{255, 150, 255, 255} // Bright pink
	colorFurnitureCheck  = color.RGBA{180, 105, 242, 255} // Violet-purple (checked; natural hue shift from pink)
	colorExitLocked      = color.RGBA{255, 100, 100, 255} // Bright red
	colorExitUnlocked    = color.RGBA{100, 255, 100, 255} // Bright green
	colorSubtle          = color.RGBA{120, 130, 180, 255} // Soft blue-purple-gray
	colorUnpoweredSubtle = color.RGBA{90, 95, 120, 255}   // Muted gray for unpowered due to dependency (room terminal off)
	colorLocation        = color.RGBA{160, 170, 210, 255} // Softer blue-gray for location/room labels
	colorPlaque          = color.RGBA{118, 112, 102, 255} // Diegetic stencil / stamped corridor signage
	colorText            = color.RGBA{200, 210, 245, 255} // Soft off-white with blue-purple tint
	colorAction          = color.RGBA{180, 150, 250, 255} // Blue-purple (less pink, more blue)
	colorDenied          = color.RGBA{255, 100, 100, 255} // Bright red
	colorPanelBackground = color.RGBA{30, 30, 50, 220}    // Semi-transparent dark
	colorFocusBackground = color.RGBA{60, 80, 100, 200}   // Cvar-backed fallback when focus plate has no opts.Color context
	// Fallback when a tile needs a “blocked” plate but no CellRenderOptions are available (should be rare).
	colorBlockedBackground = color.RGBA{100, 100, 130, 220}
	colorHazardBackground  = color.RGBA{80, 30, 30, 220} // Dark red for impassable hazards (e.g. sparks)

	// Callout colors
	ColorCalloutInfo    = color.RGBA{200, 200, 255, 255} // Light blue for info
	ColorCalloutSuccess = color.RGBA{100, 255, 150, 255} // Green for success
	ColorCalloutWarning = color.RGBA{255, 220, 100, 255} // Yellow for warnings
	ColorCalloutDanger  = color.RGBA{255, 120, 120, 255} // Red for danger/blocked
	ColorCalloutItem    = color.RGBA{210, 185, 110, 255} // Tan — item callouts (see colorItem)
	ColorCalloutKeycard = color.RGBA{100, 150, 255, 255} // Keycard pickup / map icon
	ColorCalloutBattery = color.RGBA{255, 200, 100, 255} // Battery pickup / map icon
)

// Icon constants - Unicode characters for proper font rendering
const (
	IconWall               = "▒"
	IconUnvisited          = "●"
	IconVisited            = "○"
	IconVoid               = " "
	IconExitLocked         = "▲" // Locked lift (unpowered)
	IconExitUnlocked       = "△" // Unlocked lift (powered)
	IconKey                = "K" // Keycard on floor (ASCII; Unicode ⚷ often missing in mono fallback fonts)
	IconMap                = "M" // Station map pickup (ASCII: readable in all mono fonts)
	IconItem               = "?" // Generic item on floor
	IconBattery            = "■" // Battery on floor
	IconGeneratorUnpowered = "◇" // Unpowered generator
	IconGeneratorPowered   = "◆" // Powered generator
	// Doors use Basic Latin so they render with the Go Mono fallback (Cascadia load failure)
	// and minimal fonts; geometric ▣/□ often appear as missing-glyph boxes there.
	IconDoorLocked     = "+" // Locked door
	IconDoorUnlocked   = "/" // Unlocked door
	IconTerminalUnused = "▫" // Unused CCTV terminal
	IconTerminalUsed   = "▪" // Used CCTV terminal
	IconMaintenance    = "▤" // Maintenance terminal
	IconRelayClosed    = "╬" // Corridor relay conducting
	IconRelayOpen      = "╳" // Corridor relay open (blocks grid)
)

// Floor icons for different room types (visited/unvisited pairs)
var roomFloorIcons = map[string][2]string{
	"Bridge":          {"◎", "◉"}, // Command areas
	"Command Center":  {"◎", "◉"},
	"Communications":  {"◎", "◉"},
	"Security":        {"◎", "◉"},
	"Engineering":     {"▫", "▪"}, // Technical areas
	"Reactor Core":    {"▫", "▪"},
	"Server Room":     {"▫", "▪"},
	"Maintenance Bay": {"▫", "▪"},
	"Life Support":    {"▫", "▪"},
	"Cargo Bay":       {"*", ":"}, // Storage areas (ASCII: matches mono fallback coverage)
	"Storage":         {"*", ":"},
	"Hangar":          {"*", ":"},
	"Armory":          {"*", ":"},
	"Med Bay":         {"◇", "◆"}, // Science/medical areas
	"Lab":             {"◇", "◆"},
	"Hydroponics":     {"◇", "◆"},
	"Observatory":     {"◇", "◆"},
	"Crew Quarters":   {"○", "●"}, // Living areas (using larger circles for visibility)
	"Mess Hall":       {"○", "●"},
	"Airlock":         {"╳", "╳"}, // Special areas
	"Corridor":        {"░", "░"}, // Corridors
}

// Tile size constraints
const (
	minTileSize  = 12
	maxTileSize  = 144 // Increased by 3x for higher zoom levels
	tileSizeStep = 4
	baseFontSize = 16.0 // Base font size at default tile size
)

const (
	// playerMoveDurationMs is the visual slide time per tile (see player_move.go).
	// Key repeat is matched so held WASD/arrows step once per completed slide.
	playerMoveDurationMs  = 140
	keyRepeatInitialDelay = playerMoveDurationMs
	keyRepeatInterval     = playerMoveDurationMs
)
