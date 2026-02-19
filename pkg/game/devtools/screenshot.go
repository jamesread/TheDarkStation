// Package devtools provides developer tools for testing and debugging.
package devtools

import (
	"fmt"
	"os"
	"strings"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// SaveScreenshotHTML saves the current map view as an HTML file
func SaveScreenshotHTML(g *state.Game) string {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("screenshot-%s.html", timestamp)

	viewportRows, viewportCols := renderer.GetViewportSize()

	// Calculate viewport bounds centered on player
	playerRow := g.CurrentCell.Row
	playerCol := g.CurrentCell.Col
	startRow := playerRow - viewportRows/2
	startCol := playerCol - viewportCols/2

	// Build the HTML
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>The Dark Station - Screenshot</title>
    <style>
        body {
            background-color: #1a1a2e;
            color: #eee;
            font-family: 'Courier New', monospace;
            padding: 20px;
        }
        .header {
            color: #bb86fc;
            font-size: 18px;
            margin-bottom: 10px;
        }
        .room-name {
            color: #888;
            margin-bottom: 20px;
        }
        .map-container {
            background-color: #0f0f1a;
            padding: 20px;
            border-radius: 8px;
            display: inline-block;
            margin: 20px 0;
        }
        .map-row {
            white-space: pre;
            line-height: 1.2;
            font-size: 16px;
        }
        .player { color: #00ff00; font-weight: bold; }
        .wall { color: #666; }
        .floor { color: #888; }
        .floor-visited { color: #aaa; }
        .door-locked { color: #ffff00; font-weight: bold; }
        .door-unlocked { color: #00aa00; }
        .keycard { color: #4444ff; }
        .item { color: #bb86fc; }
        .battery { color: #bb86fc; font-weight: bold; }
        .hazard { color: #ff4444; }
        .hazard-ctrl { color: #00ffff; }
        .generator-off { color: #ff4444; font-weight: bold; }
        .generator-on { color: #00aa00; }
        .terminal { color: #4444ff; }
        .terminal-used { color: #666; }
        .furniture { color: #ff66ff; font-weight: bold; }
        .furniture-checked { color: #aaaa00; }
        .exit-locked { color: #ff4444; font-weight: bold; }
        .exit-unlocked { color: #00aa00; }
        .void { color: #1a1a2e; }
        .inventory {
            margin-top: 20px;
            color: #888;
        }
        .inventory-item { color: #bb86fc; }
        .messages {
            margin-top: 20px;
            border-top: 1px solid #333;
            padding-top: 10px;
        }
        .message { color: #ccc; margin: 5px 0; }
    </style>
</head>
<body>
`)

	// Header
	html.WriteString(fmt.Sprintf(`    <div class="header">Deck %d</div>`+"\n", g.Level))
	html.WriteString(fmt.Sprintf(`    <div class="room-name">In: %s</div>`+"\n", g.CurrentCell.Name))

	// Map container
	html.WriteString(`    <div class="map-container">` + "\n")

	// Render the viewport
	for vRow := 0; vRow < viewportRows; vRow++ {
		mapRow := startRow + vRow
		html.WriteString(`        <div class="map-row">`)

		for vCol := 0; vCol < viewportCols; vCol++ {
			mapCol := startCol + vCol
			cell := g.Grid.GetCell(mapRow, mapCol)
			icon, class := getCellHTMLInfo(g, cell)
			html.WriteString(fmt.Sprintf(`<span class="%s">%s</span>`, class, icon))
		}

		html.WriteString("</div>\n")
	}

	html.WriteString(`    </div>` + "\n")

	// Inventory
	html.WriteString(`    <div class="inventory">Inventory: `)
	if g.OwnedItems.Size() == 0 && g.Batteries == 0 {
		html.WriteString(`<span style="color:#666">(empty)</span>`)
	} else {
		first := true
		g.OwnedItems.Each(func(item *world.Item) {
			if !first {
				html.WriteString(", ")
			}
			html.WriteString(fmt.Sprintf(`<span class="inventory-item">%s</span>`, item.Name))
			first = false
		})
		if g.Batteries > 0 {
			if !first {
				html.WriteString(", ")
			}
			html.WriteString(fmt.Sprintf(`<span class="battery">Batteries x%d</span>`, g.Batteries))
		}
	}
	html.WriteString(`</div>` + "\n")

	// Generator status
	if len(g.Generators) > 0 {
		html.WriteString(`    <div class="inventory">Generators: `)
		for i, gen := range g.Generators {
			if i > 0 {
				html.WriteString(", ")
			}
			if gen.IsPowered() {
				html.WriteString(fmt.Sprintf(`<span class="generator-on">#%d POWERED</span>`, i+1))
			} else {
				html.WriteString(fmt.Sprintf(`<span class="generator-off">#%d %d/%d</span>`, i+1, gen.BatteriesInserted, gen.BatteriesRequired))
			}
		}
		html.WriteString(`</div>` + "\n")
	}

	// Messages
	if len(g.Messages) > 0 {
		html.WriteString(`    <div class="messages">` + "\n")
		for _, msg := range g.Messages {
			// Strip ANSI codes for HTML output
			cleanMsg := stripANSI(msg.Text)
			html.WriteString(fmt.Sprintf(`        <div class="message">%s</div>`+"\n", cleanMsg))
		}
		html.WriteString(`    </div>` + "\n")
	}

	html.WriteString(`</body>
</html>
`)

	// Write to file
	os.WriteFile(filename, []byte(html.String()), 0644)
	return filename
}

// getCellHTMLInfo returns the icon and CSS class for a cell
func getCellHTMLInfo(g *state.Game, r *world.Cell) (string, string) {
	if r == nil {
		return " ", "void"
	}

	// Player position
	if g.CurrentCell == r {
		return "@", "player"
	}

	// Get game-specific data for this cell
	data := gameworld.GetGameData(r)

	// Hazard (show if has map or discovered)
	if gameworld.HasHazard(r) && (g.HasMap || r.Discovered) {
		if data.Hazard.IsBlocking() {
			return data.Hazard.GetIcon(), "hazard"
		}
	}

	// Hazard Control (show if has map or discovered)
	if gameworld.HasHazardControl(r) && (g.HasMap || r.Discovered) {
		if !data.HazardControl.Activated {
			return entities.GetControlIcon(data.HazardControl.Type), "hazard-ctrl"
		}
		return entities.GetControlIcon(data.HazardControl.Type), "terminal-used"
	}

	// Door (show if has map or discovered)
	if gameworld.HasDoor(r) && (g.HasMap || r.Discovered) {
		if data.Door.Locked {
			return "▣", "door-locked"
		}
		return "□", "door-unlocked"
	}

	// Generator (show if has map or discovered)
	if gameworld.HasGenerator(r) && (g.HasMap || r.Discovered) {
		if data.Generator.IsPowered() {
			return "◆", "generator-on"
		}
		return "◇", "generator-off"
	}

	// CCTV Terminal (show if has map or discovered)
	if gameworld.HasTerminal(r) && (g.HasMap || r.Discovered) {
		if data.Terminal.IsUsed() {
			return "▪", "terminal-used"
		}
		return "▫", "terminal"
	}

	// Furniture (show if has map or discovered)
	if gameworld.HasFurniture(r) && (g.HasMap || r.Discovered) {
		if data.Furniture.IsChecked() {
			return data.Furniture.Icon, "furniture-checked"
		}
		return data.Furniture.Icon, "furniture"
	}

	// Exit cell (show if has map or discovered)
	if r.ExitCell && (g.HasMap || r.Discovered) {
		if r.Locked && !g.AllGeneratorsPowered() {
			return "▲", "exit-locked"
		}
		return "△", "exit-unlocked"
	}

	// Items on floor (show if has map or discovered)
	if r.ItemsOnFloor.Size() > 0 && (g.HasMap || r.Discovered) {
		if cellHasKeycard(r) {
			return "K", "keycard"
		}
		if cellHasBattery(r) {
			return "■", "battery"
		}
		return "?", "item"
	}

	// Visited rooms
	if r.Visited {
		return getFloorIconHTML(r.Name, true), "floor-visited"
	}

	// Discovered but not visited
	if r.Discovered {
		if r.Room {
			return getFloorIconHTML(r.Name, false), "floor"
		}
		return "▒", "wall"
	}

	// Has map - show rooms faintly
	if g.HasMap && r.Room {
		return getFloorIconHTML(r.Name, false), "floor"
	}

	// Non-room cells adjacent to discovered/visited rooms render as walls
	if !r.Room && hasAdjacentDiscoveredRoomHTML(r) {
		return "▒", "wall"
	}

	// Unknown/void
	return " ", "void"
}

// getFloorIconHTML returns floor icons for HTML output
func getFloorIconHTML(roomName string, visited bool) string {
	roomFloorIcons := map[string][2]string{
		"Bridge":          {"◎", "◉"},
		"Command Center":  {"◎", "◉"},
		"Communications":  {"◎", "◉"},
		"Security":        {"◎", "◉"},
		"Engineering":     {"▫", "▪"},
		"Reactor Core":    {"▫", "▪"},
		"Server Room":     {"▫", "▪"},
		"Maintenance Bay": {"▫", "▪"},
		"Life Support":    {"▫", "▪"},
		"Cargo Bay":       {"□", "▣"},
		"Storage":         {"□", "▣"},
		"Hangar":          {"□", "▣"},
		"Armory":          {"□", "▣"},
		"Med Bay":         {"◇", "◆"},
		"Lab":             {"◇", "◆"},
		"Hydroponics":     {"◇", "◆"},
		"Observatory":     {"◇", "◆"},
		"Crew Quarters":   {"·", "•"},
		"Mess Hall":       {"·", "•"},
		"Airlock":         {"╳", "╳"},
		"Corridor":        {"░", "░"},
	}

	for baseRoom, icons := range roomFloorIcons {
		if ContainsSubstring(roomName, baseRoom) {
			if visited {
				return icons[0]
			}
			return icons[1]
		}
	}
	if visited {
		return "○"
	}
	return "●"
}

// hasAdjacentDiscoveredRoomHTML checks if any adjacent cell is a discovered or visited room
func hasAdjacentDiscoveredRoomHTML(c *world.Cell) bool {
	neighbors := []*world.Cell{c.North, c.East, c.South, c.West}
	for _, n := range neighbors {
		if n != nil && n.Room && (n.Discovered || n.Visited) {
			return true
		}
	}
	return false
}

// cellHasKeycard checks if a cell has a keycard item on the floor
func cellHasKeycard(c *world.Cell) bool {
	hasKeycard := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if ContainsSubstring(item.Name, "Keycard") || ContainsSubstring(item.Name, "keycard") {
			hasKeycard = true
		}
	})
	return hasKeycard
}

// cellHasBattery checks if a cell has a battery item on the floor
func cellHasBattery(c *world.Cell) bool {
	hasBattery := false
	c.ItemsOnFloor.Each(func(item *world.Item) {
		if ContainsSubstring(item.Name, "Battery") || ContainsSubstring(item.Name, "battery") {
			hasBattery = true
		}
	})
	return hasBattery
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
