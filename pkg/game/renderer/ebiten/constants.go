// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import "image/color"

// Color palette for the game - brighter colors for visibility
var (
	colorBackground        = color.RGBA{26, 26, 46, 255}    // Dark blue-gray
	colorMapBackground     = color.RGBA{15, 15, 26, 255}    // Darker for map area
	colorPlayer            = color.RGBA{0, 255, 0, 255}     // Bright green
	colorWall              = color.RGBA{180, 180, 200, 255} // Light gray-blue for wall text
	colorWallBg            = color.RGBA{60, 60, 80, 255}    // Darker background for walls
	colorWallBgPowered     = color.RGBA{40, 80, 40, 255}    // Dark green background for walls in powered rooms
	colorFloor             = color.RGBA{100, 100, 120, 255} // Medium gray for undiscovered
	colorFloorVisited      = color.RGBA{160, 160, 180, 255} // Lighter gray for visited
	colorDoorLocked        = color.RGBA{255, 255, 0, 255}   // Bright yellow
	colorDoorUnlocked      = color.RGBA{0, 220, 0, 255}     // Bright green
	colorKeycard           = color.RGBA{100, 150, 255, 255} // Bright blue
	colorItem              = color.RGBA{220, 170, 255, 255} // Bright purple
	colorBattery           = color.RGBA{255, 200, 100, 255} // Orange for batteries
	colorHazard            = color.RGBA{255, 80, 80, 255}   // Bright red
	colorHazardCtrl        = color.RGBA{255, 150, 200, 255} // Pink for circuit breakers
	colorGeneratorOff      = color.RGBA{255, 100, 100, 255} // Bright red
	colorGeneratorOn       = color.RGBA{0, 255, 100, 255}   // Bright green
	colorTerminal          = color.RGBA{100, 150, 255, 255} // Bright blue
	colorTerminalUsed      = color.RGBA{120, 120, 140, 255} // Medium gray
	colorMaintenance       = color.RGBA{255, 165, 0, 255}   // Orange for maintenance terminals
	colorFurniture         = color.RGBA{255, 150, 255, 255} // Bright pink
	colorFurnitureCheck    = color.RGBA{200, 180, 100, 255} // Tan/brown
	colorExitLocked        = color.RGBA{255, 100, 100, 255} // Bright red
	colorExitUnlocked      = color.RGBA{100, 255, 100, 255} // Bright green
	colorSubtle            = color.RGBA{120, 130, 180, 255} // Soft blue-purple-gray
	colorText              = color.RGBA{200, 210, 245, 255} // Soft off-white with blue-purple tint
	colorAction            = color.RGBA{180, 150, 250, 255} // Blue-purple (less pink, more blue)
	colorDenied            = color.RGBA{255, 100, 100, 255} // Bright red
	colorPanelBackground   = color.RGBA{30, 30, 50, 220}    // Semi-transparent dark
	colorFocusBackground   = color.RGBA{60, 80, 100, 200}   // Dark blue-gray for focused/interacted cell (darker than cell text)
	colorBlockedBackground = color.RGBA{100, 100, 130, 220} // Brighter background for locked doors that need to be cleared
	colorHazardBackground  = color.RGBA{80, 30, 30, 220}    // Dark red for impassable hazards (e.g. sparks)

	// Callout colors
	ColorCalloutInfo    = color.RGBA{200, 200, 255, 255} // Light blue for info
	ColorCalloutSuccess = color.RGBA{100, 255, 150, 255} // Green for success
	ColorCalloutWarning = color.RGBA{255, 220, 100, 255} // Yellow for warnings
	ColorCalloutDanger  = color.RGBA{255, 120, 120, 255} // Red for danger/blocked
	ColorCalloutItem    = color.RGBA{220, 170, 255, 255} // Purple for items
)

// Icon constants - Unicode characters for proper font rendering
const (
	PlayerIcon             = "@"
	IconWall               = "▒"
	IconUnvisited          = "●"
	IconVisited            = "○"
	IconVoid               = " "
	IconExitLocked         = "▲" // Locked lift (unpowered)
	IconExitUnlocked       = "△" // Unlocked lift (powered)
	IconKey                = "⚷" // Key item on floor
	IconItem               = "?" // Generic item on floor
	IconBattery            = "■" // Battery on floor
	IconGeneratorUnpowered = "◇" // Unpowered generator
	IconGeneratorPowered   = "◆" // Powered generator
	IconDoorLocked         = "▣" // Locked door
	IconDoorUnlocked       = "□" // Unlocked door
	IconTerminalUnused     = "▫" // Unused CCTV terminal
	IconTerminalUsed       = "▪" // Used CCTV terminal
	IconMaintenance        = "▤" // Maintenance terminal
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
	"Cargo Bay":       {"□", "▣"}, // Storage areas
	"Storage":         {"□", "▣"},
	"Hangar":          {"□", "▣"},
	"Armory":          {"□", "▣"},
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
	keyRepeatInitialDelay = 500 // Initial delay before first repeat (milliseconds)
	keyRepeatInterval     = 100 // Interval between repeat events (milliseconds)
)
