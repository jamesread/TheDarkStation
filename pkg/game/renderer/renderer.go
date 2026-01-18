package renderer

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gookit/color"
	gettext "github.com/gosexy/gettext"
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/terminal"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// Icon constants for Abandoned Station
const (
	PlayerIcon             = "@"
	IconWall               = "▒"
	IconUnvisited          = "●"
	IconVisited            = "○"
	IconVoid               = " "
	IconExitLocked         = "▲"  // Locked lift (unpowered)
	IconExitUnlocked       = "△"  // Unlocked lift (powered)
	IconKey                = "⚷"  // Key item on floor
	IconItem               = "?"  // Generic item on floor
	IconBattery            = "■"  // Battery on floor
	IconGeneratorUnpowered = "◇"  // Unpowered generator
	IconGeneratorPowered   = "◆"  // Powered generator
	IconDoorLocked         = "▣"  // Locked door
	IconDoorUnlocked       = "□"  // Unlocked door
	IconTerminalUnused     = "▫"  // Unused CCTV terminal
	IconTerminalUsed       = "▪"  // Used CCTV terminal
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
	"Crew Quarters":   {"·", "•"}, // Living areas
	"Mess Hall":       {"·", "•"},
	"Airlock":         {"╳", "╳"}, // Special areas
	"Corridor":        {"░", "░"}, // Corridors
}

// getFloorIcon returns the appropriate floor icon for a room based on its name
func getFloorIcon(roomName string, visited bool) string {
	// Check each room type to see if it's contained in the room name
	for baseRoom, icons := range roomFloorIcons {
		if strings.Contains(roomName, baseRoom) {
			if visited {
				return icons[0] // visited icon
			}
			return icons[1] // unvisited icon
		}
	}
	// Default icons
	if visited {
		return IconVisited
	}
	return IconUnvisited
}

// Viewport margins and minimum sizes
const (
	ViewportMinRows    = 7
	ViewportMinCols    = 15
	ViewportSideMargin = 26 // Space for West/East labels
	// Lines needed outside viewport:
	// - Level indicator + blank (2)
	// - Room description + blank (2)
	// - North label + blank (2)
	// - South label + blanks (3)
	// - Status bar (2-3 lines for inventory + generators)
	// - Actions (1)
	// - Messages pane (header + 5 messages + footer = 7)
	// - Input prompt (2)
	ViewportTopMargin = 24
)

// GetViewportSize returns the viewport dimensions based on terminal size
func GetViewportSize() (rows, cols int) {
	termWidth, termHeight := terminal.GetSize()

	// Calculate available space
	cols = termWidth - (ViewportSideMargin * 2)
	rows = termHeight - ViewportTopMargin

	// Ensure minimum size
	if cols < ViewportMinCols {
		cols = ViewportMinCols
	}
	if rows < ViewportMinRows {
		rows = ViewportMinRows
	}

	// Keep rows odd for centering
	if rows%2 == 0 {
		rows--
	}
	// Keep cols odd for centering
	if cols%2 == 0 {
		cols--
	}

	return rows, cols
}

var (
	ColorCell        color.Style
	ColorCellText    color.Style
	ColorAction      color.Style
	ColorActionShort color.Style
	ColorDenied      color.Style
	ColorItem        color.Style
	ColorSubtle      color.Style
	ColorPlayer      color.Style
	ColorExitOpen    color.Style
	ColorDoor        color.Style     // Generic door color
	ColorKeycard     color.Style     // Keycard color
	ColorFurniture   color.Style     // Furniture color

	regexpStringFunctions *regexp.Regexp
)

// InitColors initializes the color styles
func InitColors() {
	ColorCell = color.Style{color.FgGray}
	ColorCellText = color.Style{color.FgBlue}
	ColorAction = color.Style{color.FgMagenta}
	ColorActionShort = color.Style{color.FgMagenta, color.OpBold}
	ColorDenied = color.Style{color.FgRed, color.OpBold}
	ColorItem = color.Style{color.FgGreen, color.OpBold}
	ColorSubtle = color.Style{color.FgGray, color.OpBold}
	ColorPlayer = color.Style{color.FgGreen, color.BgBlack, color.OpBold}
	ColorExitOpen = color.Style{color.FgGreen}            // Dark green (no bold)
	ColorDoor = color.Style{color.FgYellow, color.OpBold} // Yellow for doors
	ColorKeycard = color.Style{color.FgCyan, color.OpBold} // Cyan for keycards
	ColorFurniture = color.Style{color.FgWhite}           // White for furniture

	regexpStringFunctions = regexp.MustCompile(`([a-zA-Z_]*){([a-z A-Z0-9_,:]+)}`)
}

// FormatString formats a string with special markup
func FormatString(msg string, a ...any) string {
	ret := fmt.Sprintf(msg, a...)

	matches := regexpStringFunctions.FindAllStringSubmatch(ret, -1)

	for _, match := range matches {
		function := match[1]
		operand := match[2]

		val := "blat"

		switch function {
		case "GT":
			val = gettext.Gettext(operand)
		case "ITEM":
			val = ColorItem.Sprintf(operand)
		case "ROOM":
			val = ColorCell.Sprintf(gettext.Gettext(operand))
		case "ACTION":
			val = ColorActionShort.Sprintf(operand[0:1]) + ColorAction.Sprintf(operand[1:])
		default:
			ret = fmt.Sprintf("ERROR, function not found: %v -> %v", function, operand)
		}

		ret = strings.Replace(ret, match[0], val, -1)
	}

	return ret
}

// PrintString prints a formatted string
func PrintString(msg string, a ...any) {
	fmt.Print(FormatString(msg, a...))
}

// PrintStringCenter prints a string centered
func PrintStringCenter(s string) {
	w := 28 + (len(s) - len(color.ClearCode(s)))
	PrintString("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

// PrintBullet prints a bulleted item
func PrintBullet(txt string) {
	fmt.Printf("- " + FormatString(txt) + "\n")
}

// Clear clears the terminal screen
func Clear() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

// RenderCell returns the string representation of a cell
func RenderCell(g *state.Game, r *world.Cell) string {
	if r == nil {
		return IconVoid
	}

	// Player position
	if g.CurrentCell == r {
		return ColorPlayer.Sprint(PlayerIcon)
	}

	// Door (show if has map or discovered)
	if r.HasDoor() && (g.HasMap || r.Discovered) {
		if r.Door.Locked {
			return ColorDoor.Sprintf(IconDoorLocked)
		}
		return ColorExitOpen.Sprintf(IconDoorUnlocked)
	}

	// Generator (show if has map or discovered)
	if r.HasGenerator() && (g.HasMap || r.Discovered) {
		if r.Generator.IsPowered() {
			return ColorExitOpen.Sprintf(IconGeneratorPowered) // Dark green when powered
		}
		return ColorDenied.Sprintf(IconGeneratorUnpowered)
	}

	// CCTV Terminal (show if has map or discovered)
	if r.HasTerminal() && (g.HasMap || r.Discovered) {
		if r.Terminal.IsUsed() {
			return ColorSubtle.Sprintf(IconTerminalUsed)
		}
		return ColorCellText.Sprintf(IconTerminalUnused)
	}

	// Furniture (show if has map or discovered)
	if r.HasFurniture() && (g.HasMap || r.Discovered) {
		return ColorFurniture.Sprintf(r.Furniture.Icon)
	}

	// Exit cell (show if has map or discovered)
	if r.ExitCell && (g.HasMap || r.Discovered) {
		// Exit is red if locked (generators not all powered), dark green if unlocked/powered
		if r.Locked && !g.AllGeneratorsPowered() {
			return ColorDenied.Sprintf(IconExitLocked)
		}
		return ColorExitOpen.Sprintf(IconExitUnlocked)
	}

	// Items on floor (show if has map or discovered) - keycards/batteries get special icons
	if r.ItemsOnFloor.Size() > 0 && (g.HasMap || r.Discovered) {
		if cellHasKeycard(r) {
			return ColorKeycard.Sprintf(IconKey)
		}
		if cellHasBattery(r) {
			return ColorAction.Sprintf(IconBattery)
		}
		return ColorItem.Sprintf(IconItem)
	}

	// Visited rooms
	if r.Visited {
		return ColorCell.Sprintf(getFloorIcon(r.Name, true))
	}

	// Discovered but not visited
	if r.Discovered {
		if r.Room {
			return ColorSubtle.Sprintf(getFloorIcon(r.Name, false))
		}
		return ColorSubtle.Sprintf(IconWall)
	}

	// Has map - show rooms faintly
	if g.HasMap && r.Room {
		return ColorSubtle.Sprintf(getFloorIcon(r.Name, false))
	}

	// Non-room cells adjacent to discovered/visited rooms render as walls
	// This ensures perimeter cells show as walls when you can see them
	if !r.Room && hasAdjacentDiscoveredRoom(r) {
		return ColorSubtle.Sprintf(IconWall)
	}

	// Unknown/void
	return IconVoid
}

// cellHasKeycard checks if a cell has a keycard item on the floor
func cellHasKeycard(c *world.Cell) bool {
	hasKeycard := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if strings.Contains(strings.ToLower(item.Name), "keycard") {
			hasKeycard = true
		}
	})
	return hasKeycard
}

// cellHasBattery checks if a cell has a battery item on the floor
func cellHasBattery(c *world.Cell) bool {
	hasBattery := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if strings.Contains(strings.ToLower(item.Name), "battery") {
			hasBattery = true
		}
	})
	return hasBattery
}

// hasAdjacentDiscoveredRoom checks if any adjacent cell is a discovered or visited room
func hasAdjacentDiscoveredRoom(c *world.Cell) bool {
	neighbors := []*world.Cell{c.North, c.East, c.South, c.West}
	for _, n := range neighbors {
		if n != nil && n.Room && (n.Discovered || n.Visited) {
			return true
		}
	}
	return false
}

// SetJoin joins item names from a set with commas
func SetJoin(set *world.ItemSet) string {
	ret := ""

	set.Each(func(i *world.Item) {
		ret += i.Name + ","
	})

	// Remove trailing comma
	if len(ret) > 0 {
		ret = ret[:len(ret)-1]
	}

	return ret
}

// CanEnterCell checks if the player can enter a cell (without logging)
func CanEnterCell(g *state.Game, r *world.Cell) (bool, *world.ItemSet) {
	missingItems := mapset.New[*world.Item]()

	if r == nil || !r.Room {
		return false, &missingItems
	}

	r.RequiredItems.Each(func(reqItem *world.Item) {
		if !g.OwnedItems.Has(reqItem) {
			missingItems.Put(reqItem)
		}
	})

	return missingItems.Size() == 0, &missingItems
}

// GetDirectionActionText returns the action text for a direction
func GetDirectionActionText(g *state.Game, c *world.Cell, direction string) string {
	if c == nil || !c.Room {
		return ColorSubtle.Sprintf("# Wall #")
	}

	lockedText := ""

	if canEnter, missingItems := CanEnterCell(g, c); !canEnter {
		lockedText = ColorDenied.Sprintf(" (%v)", SetJoin(missingItems))
	}

	// Get the display key based on navigation style
	displayKey := direction
	if g.NavStyle == state.NavStyleVim {
		switch direction {
		case "North":
			displayKey = "k"
		case "South":
			displayKey = "j"
		case "East":
			displayKey = "l"
		case "West":
			displayKey = "h"
		}
	}

	return fmt.Sprintf("ACTION{%v}%v", displayKey, lockedText)
}

// PrintMap renders the game map
func PrintMap(g *state.Game) {
	termWidth := terminal.GetWidth()
	viewportRows, viewportCols := GetViewportSize()

	// Calculate indent to center the map
	// West label area (24 chars) + map (viewportCols) + East label area (24 chars)
	westLabelWidth := 24
	totalMapWidth := westLabelWidth + viewportCols + westLabelWidth
	centerIndent := (termWidth - totalMapWidth) / 2
	if centerIndent < 0 {
		centerIndent = 0
	}
	indent := strings.Repeat(" ", centerIndent+westLabelWidth)

	// Calculate indent to center North/South labels over the map viewport
	mapStartCol := centerIndent + westLabelWidth

	// Calculate viewport bounds centered on player
	playerRow := g.CurrentCell.Row
	playerCol := g.CurrentCell.Col

	// Calculate the top-left corner of the viewport
	startRow := playerRow - viewportRows/2
	startCol := playerCol - viewportCols/2

	// Print North direction label (centered over map)
	northText := FormatString(GetDirectionActionText(g, g.CurrentCell.North, "North"))
	northLen := len(color.ClearCode(northText))
	northIndent := mapStartCol + (viewportCols-northLen)/2
	if northIndent < 0 {
		northIndent = 0
	}
	fmt.Print(strings.Repeat(" ", northIndent))
	fmt.Println(northText)
	fmt.Println("")

	// Render the viewport
	for vRow := 0; vRow < viewportRows; vRow++ {
		mapRow := startRow + vRow

		// Print West label on the middle row
		if vRow == viewportRows/2 {
			txt := FormatString(GetDirectionActionText(g, g.CurrentCell.West, "West"))
			labelLen := len(color.ClearCode(txt))
			padding := centerIndent + westLabelWidth - labelLen
			if padding > 0 {
				fmt.Print(strings.Repeat(" ", padding))
			}
			fmt.Print(txt)
		} else {
			fmt.Print(indent)
		}

		// Render cells in this row
		for vCol := 0; vCol < viewportCols; vCol++ {
			mapCol := startCol + vCol

			// Check if this position is within the actual grid
			cell := g.Grid.GetCell(mapRow, mapCol)
			if cell == nil {
				fmt.Printf(ColorSubtle.Sprintf(" "))
			} else {
				fmt.Print(RenderCell(g, cell))
			}
		}

		// Print East label on the middle row
		if vRow == viewportRows/2 {
			PrintString(" %s", GetDirectionActionText(g, g.CurrentCell.East, "East"))
		}

		fmt.Print("\n")
	}

	fmt.Println("")

	// Print South direction label (centered over map)
	southText := FormatString(GetDirectionActionText(g, g.CurrentCell.South, "South"))
	southLen := len(color.ClearCode(southText))
	southIndent := mapStartCol + (viewportCols-southLen)/2
	if southIndent < 0 {
		southIndent = 0
	}
	fmt.Print(strings.Repeat(" ", southIndent))
	fmt.Println(southText)

	fmt.Println("")
}

// PrintPossibleActions prints the available actions
func PrintPossibleActions() {
	PrintBullet("ACTION{?}: \tShow hint")
}

// PrintStatusBar renders the inventory status bar
func PrintStatusBar(g *state.Game) {
	fmt.Println()

	// Show items
	fmt.Print(ColorSubtle.Sprint("Inventory: "))
	if g.OwnedItems.Size() == 0 && g.Batteries == 0 {
		fmt.Println(ColorSubtle.Sprint("(empty)"))
	} else {
		items := []string{}
		g.OwnedItems.Each(func(item *world.Item) {
			items = append(items, ColorItem.Sprint(item.Name))
		})
		// Add batteries to display
		if g.Batteries > 0 {
			items = append(items, ColorAction.Sprintf("Batteries x%d", g.Batteries))
		}
		fmt.Println(strings.Join(items, ColorSubtle.Sprint(", ")))
	}

	// Show generator status if there are generators on this level
	if len(g.Generators) > 0 {
		fmt.Print(ColorSubtle.Sprint("Generators: "))
		genStatus := []string{}
		for i, gen := range g.Generators {
			if gen.IsPowered() {
				genStatus = append(genStatus, ColorItem.Sprintf("#%d POWERED", i+1))
			} else {
				genStatus = append(genStatus, ColorDenied.Sprintf("#%d %d/%d", i+1, gen.BatteriesInserted, gen.BatteriesRequired))
			}
		}
		fmt.Println(strings.Join(genStatus, ColorSubtle.Sprint(", ")))
	}
}

// PrintMessagesPane renders the messages log pane
func PrintMessagesPane(g *state.Game) {
	width := terminal.GetWidth()

	// Create a horizontal line spanning the terminal width
	// "Messages" label is 8 chars, plus 2 spaces = 10, so we need (width - 10) / 2 dashes on each side
	label := " Messages "
	labelLen := len(label)
	sideLen := (width - labelLen) / 2
	if sideLen < 1 {
		sideLen = 1
	}

	leftDashes := strings.Repeat("─", sideLen)
	rightDashes := strings.Repeat("─", width-sideLen-labelLen)

	fmt.Println()
	fmt.Println(ColorSubtle.Sprint(leftDashes + label + rightDashes))

	if len(g.Messages) == 0 {
		fmt.Println(ColorSubtle.Sprint("  (no messages)"))
	} else {
		for _, msg := range g.Messages {
			fmt.Printf("  %s\n", msg)
		}
	}

	fmt.Println(ColorSubtle.Sprint(strings.Repeat("─", width)))
}
