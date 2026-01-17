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

	"darkcastle/pkg/engine/terminal"
	"darkcastle/pkg/engine/world"
	"darkcastle/pkg/game/state"
)

// Icon constants for The Dark Castle
const (
	PlayerIcon       = "@"
	IconWall         = "▒"
	IconUnvisited    = "●"
	IconVisited      = "○"
	IconVoid         = " "
	IconExitLocked   = "▣"  // Locked exit (closed/blocked)
	IconExitUnlocked = "⌂"  // Unlocked exit (house symbol)
	IconKey          = "⚷"  // Key item on floor
	IconItem         = "?"  // Generic item on floor
)

// Viewport dimensions (player will be centered)
const (
	ViewportRows = 9
	ViewportCols = 21
)

var (
	ColorCell        color.Style
	ColorCellText    color.Style
	ColorAction      color.Style
	ColorActionShort color.Style
	ColorDenied      color.Style
	ColorItem        color.Style
	ColorSubtle      color.Style
	ColorPlayer      color.Style

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

	// Exit cell (show if has map or discovered)
	if r.ExitCell && (g.HasMap || r.Discovered) {
		if r.Locked {
			return ColorDenied.Sprintf(IconExitLocked)
		}
		return ColorItem.Sprintf(IconExitUnlocked)
	}

	// Items on floor (show if has map or discovered) - keys get special icon
	if r.ItemsOnFloor.Size() > 0 && (g.HasMap || r.Discovered) {
		if cellHasKey(r) {
			return ColorItem.Sprintf(IconKey)
		}
		return ColorItem.Sprintf(IconItem)
	}

	// Visited rooms
	if r.Visited {
		return ColorCell.Sprintf(IconVisited)
	}

	// Discovered but not visited
	if r.Discovered {
		if r.Room {
			return ColorSubtle.Sprintf(IconUnvisited)
		}
		return ColorSubtle.Sprintf(IconWall)
	}

	// Has map - show rooms faintly
	if g.HasMap && r.Room {
		return ColorSubtle.Sprintf(IconUnvisited)
	}

	// Non-room cells adjacent to discovered/visited rooms render as walls
	// This ensures perimeter cells show as walls when you can see them
	if !r.Room && hasAdjacentDiscoveredRoom(r) {
		return ColorSubtle.Sprintf(IconWall)
	}

	// Unknown/void
	return IconVoid
}

// cellHasKey checks if a cell has a key item on the floor
func cellHasKey(c *world.Cell) bool {
	hasKey := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if strings.Contains(strings.ToLower(item.Name), "key") {
			hasKey = true
		}
	})
	return hasKey
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

	return fmt.Sprintf("ACTION{%v}: (%v) %v", displayKey, ColorCell.Sprintf(c.Name), lockedText)
}

// PrintMap renders the game map
func PrintMap(g *state.Game) {
	termWidth := terminal.GetWidth()

	// Calculate indent to center the map
	// West label area (24 chars) + map (ViewportCols) + East label area (24 chars)
	westLabelWidth := 24
	totalMapWidth := westLabelWidth + ViewportCols + westLabelWidth
	centerIndent := (termWidth - totalMapWidth) / 2
	if centerIndent < 0 {
		centerIndent = 0
	}
	indent := strings.Repeat(" ", centerIndent+westLabelWidth)

	// Calculate viewport bounds centered on player
	playerRow := g.CurrentCell.Row
	playerCol := g.CurrentCell.Col

	// Calculate the top-left corner of the viewport
	startRow := playerRow - ViewportRows/2
	startCol := playerCol - ViewportCols/2

	// Print North direction label
	fmt.Print(indent)
	PrintStringCenter(GetDirectionActionText(g, g.CurrentCell.North, "North"))
	fmt.Println("")
	fmt.Println("")

	// Render the viewport
	for vRow := 0; vRow < ViewportRows; vRow++ {
		mapRow := startRow + vRow

		// Print West label on the middle row
		if vRow == ViewportRows/2 {
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
		for vCol := 0; vCol < ViewportCols; vCol++ {
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
		if vRow == ViewportRows/2 {
			PrintString(" %s", GetDirectionActionText(g, g.CurrentCell.East, "East"))
		}

		fmt.Print("\n")
	}

	fmt.Println("")

	// Print South direction label
	fmt.Print(indent)
	PrintStringCenter(GetDirectionActionText(g, g.CurrentCell.South, "South"))

	fmt.Println("")
	fmt.Println("")
}

// PrintPossibleActions prints the available actions
func PrintPossibleActions() {
	PrintBullet("ACTION{?}: \tShow hint")
}

// PrintStatusBar renders the inventory status bar
func PrintStatusBar(g *state.Game) {
	fmt.Println()
	fmt.Print(ColorSubtle.Sprint("Inventory: "))

	if g.OwnedItems.Size() == 0 {
		fmt.Println(ColorSubtle.Sprint("(empty)"))
	} else {
		items := []string{}
		g.OwnedItems.Each(func(item *world.Item) {
			items = append(items, ColorItem.Sprint(item.Name))
		})
		fmt.Println(strings.Join(items, ColorSubtle.Sprint(", ")))
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
