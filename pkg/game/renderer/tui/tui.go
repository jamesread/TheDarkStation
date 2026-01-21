package tui

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gookit/color"
	"github.com/leonelquinteros/gotext"

	"darkstation/pkg/engine/input"
	"darkstation/pkg/engine/terminal"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// Icon constants for Abandoned Station
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

// dynamicGet is used for runtime translation key lookups.
// We use a function variable to avoid go vet's non-constant format string check,
// since we intentionally look up translation keys dynamically from markup.
var dynamicGet = gotext.Get

// TUIRenderer is the terminal-based renderer implementation
type TUIRenderer struct {
	colorCell           color.Style
	colorCellText       color.Style
	colorAction         color.Style
	colorActionShort    color.Style
	colorDenied         color.Style
	colorItem           color.Style
	colorSubtle         color.Style
	colorPlayer         color.Style
	colorExitOpen       color.Style
	colorDoor           color.Style
	colorKeycard        color.Style
	colorFurniture      color.Style
	colorFurnitureCheck color.Style
	colorHazard         color.Style
	colorHazardCtrl     color.Style

	regexpStringFunctions *regexp.Regexp
}

// New creates a new TUI renderer
func New() *TUIRenderer {
	return &TUIRenderer{}
}

// Init initializes the TUI renderer (colors, etc.)
func (t *TUIRenderer) Init() {
	t.colorCell = color.Style{color.FgGray}
	t.colorCellText = color.Style{color.FgBlue}
	t.colorAction = color.Style{color.FgMagenta}
	t.colorActionShort = color.Style{color.FgMagenta, color.OpBold}
	t.colorDenied = color.Style{color.FgRed, color.OpBold}
	t.colorItem = color.Style{color.FgMagenta} // Dark purple for inventory items
	t.colorSubtle = color.Style{color.FgGray, color.OpBold}
	t.colorPlayer = color.Style{color.FgGreen, color.BgBlack, color.OpBold}
	t.colorExitOpen = color.Style{color.FgGreen}                  // Dark green (no bold)
	t.colorDoor = color.Style{color.FgYellow, color.OpBold}       // Yellow for doors
	t.colorKeycard = color.Style{color.FgBlue}                    // Dark blue for keycards
	t.colorFurniture = color.Style{color.FgMagenta, color.OpBold} // Pink for unchecked furniture
	t.colorFurnitureCheck = color.Style{color.FgYellow}           // Brown/dark yellow for checked furniture
	t.colorHazard = color.Style{color.FgRed}                      // Red for hazards
	t.colorHazardCtrl = color.Style{color.FgCyan}                 // Cyan for hazard controls

	t.regexpStringFunctions = regexp.MustCompile(`([a-zA-Z_]*){([a-z A-Z0-9_,:]+)}`)
}

// Clear clears the terminal screen
func (t *TUIRenderer) Clear() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

// GetInput gets user input from the terminal and returns a high-level Intent.
func (t *TUIRenderer) GetInput() input.Intent {
	raw := input.RawInput{
		Device: input.DeviceTerminal,
		Code:   input.GetInputWithArrows(),
		// Timestamp left zero for now; terminal input is inherently low frequency.
	}
	debounced := input.NewDebouncedInput(raw)
	return input.MapToIntent(debounced)
}

// StyleText applies a style to text
func (t *TUIRenderer) StyleText(text string, style renderer.TextStyle) string {
	switch style {
	case renderer.StyleCell:
		return t.colorCell.Sprint(text)
	case renderer.StyleCellText:
		return t.colorCellText.Sprint(text)
	case renderer.StyleItem:
		return t.colorItem.Sprint(text)
	case renderer.StyleAction:
		return t.colorAction.Sprint(text)
	case renderer.StyleActionShort:
		return t.colorActionShort.Sprint(text)
	case renderer.StyleDenied:
		return t.colorDenied.Sprint(text)
	case renderer.StyleKeycard:
		return t.colorKeycard.Sprint(text)
	case renderer.StyleDoor:
		return t.colorDoor.Sprint(text)
	case renderer.StyleHazard:
		return t.colorHazard.Sprint(text)
	case renderer.StyleHazardCtrl:
		return t.colorHazardCtrl.Sprint(text)
	case renderer.StyleFurniture:
		return t.colorFurniture.Sprint(text)
	case renderer.StyleFurnitureChecked:
		return t.colorFurnitureCheck.Sprint(text)
	case renderer.StyleSubtle:
		return t.colorSubtle.Sprint(text)
	case renderer.StylePlayer:
		return t.colorPlayer.Sprint(text)
	case renderer.StyleExitOpen:
		return t.colorExitOpen.Sprint(text)
	default:
		return text
	}
}

// FormatText formats a message with the markup system
func (t *TUIRenderer) FormatText(msg string, args ...any) string {
	ret := fmt.Sprintf(msg, args...)

	matches := t.regexpStringFunctions.FindAllStringSubmatch(ret, -1)

	for _, match := range matches {
		function := match[1]
		operand := match[2]

		val := "blat"

		switch function {
		case "GT":
			val = dynamicGet(operand)
		case "ITEM":
			val = t.colorItem.Sprint(operand)
		case "ROOM":
			val = t.colorCell.Sprint(dynamicGet(operand))
		case "ACTION":
			val = t.colorActionShort.Sprint(operand[0:1]) + t.colorAction.Sprint(operand[1:])
		case "FURNITURE":
			// FURNITURE{} uses the furniture checked color (tan/brown)
			val = t.colorFurnitureCheck.Sprint(operand)
		default:
			ret = fmt.Sprintf("ERROR, function not found: %v -> %v", function, operand)
		}

		ret = strings.Replace(ret, match[0], val, -1)
	}

	return ret
}

// ShowMessage displays a message to the user
func (t *TUIRenderer) ShowMessage(msg string) {
	fmt.Println(msg)
}

// GetViewportSize returns the viewport dimensions based on terminal size
func (t *TUIRenderer) GetViewportSize() (rows, cols int) {
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

// RenderFrame renders a complete game frame
func (t *TUIRenderer) RenderFrame(g *state.Game) {
	// Level indicator in top left
	t.colorAction.Printf("Deck %d\n\n", g.Level)

	// Room name
	t.printString("GT{IN_ROOM} ROOM{%v}\n\n", g.CurrentCell.Name)

	// Render the map
	t.printMap(g)

	// Status bar
	t.printStatusBar(g)

	// Actions
	t.printPossibleActions()

	// Messages pane
	t.printMessagesPane(g)

	// Input prompt
	fmt.Printf("\n> ")
}

// printString prints a formatted string
func (t *TUIRenderer) printString(msg string, a ...any) {
	fmt.Print(t.FormatText(msg, a...))
}

// printStringCenter prints a string centered
func (t *TUIRenderer) printStringCenter(s string) {
	w := 28 + (len(s) - len(color.ClearCode(s)))
	t.printString("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

// printBullet prints a bulleted item
func (t *TUIRenderer) printBullet(txt string) {
	fmt.Print("- " + t.FormatText("%s", txt) + "\n")
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

// renderCell returns the string representation of a cell
func (t *TUIRenderer) renderCell(g *state.Game, r *world.Cell) string {
	if r == nil {
		return IconVoid
	}

	// Player position
	if g.CurrentCell == r {
		return t.colorPlayer.Sprint(PlayerIcon)
	}

	// Get game-specific data for this cell
	data := gameworld.GetGameData(r)

	// Hazard (show if has map or discovered)
	if gameworld.HasHazard(r) && (g.HasMap || r.Discovered) {
		if data.Hazard.IsBlocking() {
			return t.colorHazard.Sprint(data.Hazard.GetIcon())
		}
		// Fixed hazards show as normal floor
	}

	// Hazard Control (show if has map or discovered)
	if gameworld.HasHazardControl(r) && (g.HasMap || r.Discovered) {
		if !data.HazardControl.Activated {
			return t.colorHazardCtrl.Sprint(entities.GetControlIcon(data.HazardControl.Type))
		}
		return t.colorSubtle.Sprint(entities.GetControlIcon(data.HazardControl.Type))
	}

	// Door (show if has map or discovered)
	if gameworld.HasDoor(r) && (g.HasMap || r.Discovered) {
		if data.Door.Locked {
			return t.colorDoor.Sprint(IconDoorLocked)
		}
		return t.colorExitOpen.Sprint(IconDoorUnlocked)
	}

	// Generator (show if has map or discovered)
	if gameworld.HasGenerator(r) && (g.HasMap || r.Discovered) {
		if data.Generator.IsPowered() {
			return t.colorExitOpen.Sprint(IconGeneratorPowered) // Dark green when powered
		}
		return t.colorDenied.Sprint(IconGeneratorUnpowered)
	}

	// CCTV Terminal (show if has map or discovered)
	if gameworld.HasTerminal(r) && (g.HasMap || r.Discovered) {
		if data.Terminal.IsUsed() {
			return t.colorSubtle.Sprint(IconTerminalUsed)
		}
		return t.colorCellText.Sprint(IconTerminalUnused)
	}

	// Furniture (show if has map or discovered)
	if gameworld.HasFurniture(r) && (g.HasMap || r.Discovered) {
		if data.Furniture.IsChecked() {
			return t.colorFurnitureCheck.Sprint(data.Furniture.Icon)
		}
		return t.colorFurniture.Sprint(data.Furniture.Icon)
	}

	// Exit cell (show if has map or discovered)
	if r.ExitCell && (g.HasMap || r.Discovered) {
		// Exit is red if locked (generators not all powered), dark green if unlocked/powered
		if r.Locked && !g.AllGeneratorsPowered() {
			return t.colorDenied.Sprint(IconExitLocked)
		}
		return t.colorExitOpen.Sprint(IconExitUnlocked)
	}

	// Items on floor (show if has map or discovered) - keycards/batteries get special icons
	if r.ItemsOnFloor.Size() > 0 && (g.HasMap || r.Discovered) {
		if cellHasKeycard(r) {
			return t.colorKeycard.Sprint(IconKey)
		}
		if cellHasBattery(r) {
			return t.colorAction.Sprint(IconBattery)
		}
		return t.colorItem.Sprint(IconItem)
	}

	// Visited rooms
	if r.Visited {
		return t.colorCell.Sprint(getFloorIcon(r.Name, true))
	}

	// Discovered but not visited
	if r.Discovered {
		if r.Room {
			return t.colorSubtle.Sprint(getFloorIcon(r.Name, false))
		}
		return t.colorSubtle.Sprint(IconWall)
	}

	// Has map - show rooms faintly
	if g.HasMap && r.Room {
		return t.colorSubtle.Sprint(getFloorIcon(r.Name, false))
	}

	// Non-room cells adjacent to discovered/visited rooms render as walls
	// This ensures perimeter cells show as walls when you can see them
	if !r.Room && hasAdjacentDiscoveredRoom(r) {
		return t.colorSubtle.Sprint(IconWall)
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

// getDirectionActionText returns the action text for a direction
func (t *TUIRenderer) getDirectionActionText(g *state.Game, c *world.Cell, direction string) string {
	if c == nil || !c.Room {
		return t.colorSubtle.Sprintf("# Wall #")
	}

	lockedText := ""

	if canEnter, missingItems := t.canEnterCell(g, c); !canEnter {
		lockedText = t.colorDenied.Sprintf(" (%v)", t.setJoin(missingItems))
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

// canEnterCell checks if the player can enter a cell (without logging)
func (t *TUIRenderer) canEnterCell(g *state.Game, r *world.Cell) (bool, *world.ItemSet) {
	missingItems := world.NewItemSet()

	if r == nil || !r.Room {
		return false, missingItems
	}

	r.RequiredItems.Each(func(reqItem *world.Item) {
		if !g.OwnedItems.Has(reqItem) {
			missingItems.Put(reqItem)
		}
	})

	return missingItems.Size() == 0, missingItems
}

// setJoin joins item names from a set with commas
func (t *TUIRenderer) setJoin(set *world.ItemSet) string {
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

// printMap renders the game map
func (t *TUIRenderer) printMap(g *state.Game) {
	termWidth := terminal.GetWidth()
	viewportRows, viewportCols := t.GetViewportSize()

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
	northText := t.FormatText("%s", t.getDirectionActionText(g, g.CurrentCell.North, "North"))
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
			txt := t.FormatText("%s", t.getDirectionActionText(g, g.CurrentCell.West, "West"))
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
				fmt.Print(t.colorSubtle.Sprint(" "))
			} else {
				fmt.Print(t.renderCell(g, cell))
			}
		}

		// Print East label on the middle row
		if vRow == viewportRows/2 {
			t.printString(" %s", t.getDirectionActionText(g, g.CurrentCell.East, "East"))
		}

		fmt.Print("\n")
	}

	fmt.Println("")

	// Print South direction label (centered over map)
	southText := t.FormatText("%s", t.getDirectionActionText(g, g.CurrentCell.South, "South"))
	southLen := len(color.ClearCode(southText))
	southIndent := mapStartCol + (viewportCols-southLen)/2
	if southIndent < 0 {
		southIndent = 0
	}
	fmt.Print(strings.Repeat(" ", southIndent))
	fmt.Println(southText)

	fmt.Println("")
}

// printPossibleActions prints the available actions
func (t *TUIRenderer) printPossibleActions() {
	t.printBullet("ACTION{?}: \tShow hint")
}

// printStatusBar renders the inventory status bar
func (t *TUIRenderer) printStatusBar(g *state.Game) {
	fmt.Println()

	// Show items
	fmt.Print(t.colorSubtle.Sprint("Inventory: "))
	if g.OwnedItems.Size() == 0 && g.Batteries == 0 {
		fmt.Println(t.colorSubtle.Sprint("(empty)"))
	} else {
		items := []string{}
		g.OwnedItems.Each(func(item *world.Item) {
			items = append(items, t.colorItem.Sprint(item.Name))
		})
		// Add batteries to display
		if g.Batteries > 0 {
			items = append(items, t.colorAction.Sprintf("Batteries x%d", g.Batteries))
		}
		fmt.Println(strings.Join(items, t.colorSubtle.Sprint(", ")))
	}

	// Show generator status if there are generators on this level
	if len(g.Generators) > 0 {
		fmt.Print(t.colorSubtle.Sprint("Generators: "))
		genStatus := []string{}
		for i, gen := range g.Generators {
			if gen.IsPowered() {
				genStatus = append(genStatus, t.colorItem.Sprintf("#%d POWERED", i+1))
			} else {
				genStatus = append(genStatus, t.colorDenied.Sprintf("#%d %d/%d", i+1, gen.BatteriesInserted, gen.BatteriesRequired))
			}
		}
		fmt.Println(strings.Join(genStatus, t.colorSubtle.Sprint(", ")))
	}
}

// printMessagesPane renders the messages log pane
func (t *TUIRenderer) printMessagesPane(g *state.Game) {
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
	fmt.Println(t.colorSubtle.Sprint(leftDashes + label + rightDashes))

	if len(g.Messages) == 0 {
		fmt.Println(t.colorSubtle.Sprint("  (no messages)"))
	} else {
		for _, msg := range g.Messages {
			fmt.Printf("  %s\n", msg)
		}
	}

	fmt.Println(t.colorSubtle.Sprint(strings.Repeat("─", width)))
}
