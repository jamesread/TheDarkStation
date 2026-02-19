// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/leonelquinteros/gotext"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/devtools"
	"darkstation/pkg/game/entities"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/state"
)

// ProcessIntent handles a high-level input intent from the tiered input system.
func ProcessIntent(g *state.Game, intent engineinput.Intent) {
	// Completion screen: any key returns to title
	if g.GameComplete {
		if intent.Action != engineinput.ActionNone {
			g.QuitToTitle = true
		}
		return
	}

	switch intent.Action {
	case engineinput.ActionNone:
		return

	case engineinput.ActionOpenMenu:
		RunGameplayMenu(g)
		return

	case engineinput.ActionHint:
		idx := rand.Intn(len(g.Hints))
		logMessage(g, "%s", g.Hints[idx])
		return

	case engineinput.ActionQuit:
		fmt.Println(gotext.Get("GOODBYE"))
		os.Exit(0)

	case engineinput.ActionScreenshot:
		filename := devtools.SaveScreenshotHTML(g)
		logMessage(g, "Screenshot saved to ITEM{%s}", filename)
		return

	case engineinput.ActionDevMap:
		devtools.SwitchToDevMap(g)
		return

	case engineinput.ActionDebugMapDump:
		path, err := devtools.DumpRevealedMapToFile(g)
		if err != nil {
			logMessage(g, "Map dump failed: %v", err)
		} else {
			logMessage(g, "Map dumped to ITEM{%s}", path)
		}
		return

	case engineinput.ActionResetLevel:
		ResetLevel(g)
		return

	case engineinput.ActionMoveEast:
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.East)
		return

	case engineinput.ActionMoveWest:
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.West)
		return

	case engineinput.ActionMoveNorth:
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.North)
		return

	case engineinput.ActionMoveSouth:
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.South)
		return

	case engineinput.ActionInteract:
		// Check for adjacent interactables in NSEW priority order, cycling through them
		interacted := CheckAdjacentInteractables(g)
		if !interacted {
			logMessage(g, "Nothing to interact with here.")
		}
		return
	}

	logMessage(g, "%s", gotext.Get("UNKNOWN_COMMAND"))
}

// RunGameplayMenu presents the gameplay menu with options for bindings and quitting to title.
func RunGameplayMenu(g *state.Game) {
	handler := gamemenu.NewGameplayMenuHandler()
	items := handler.GetMenuItems()
	gamemenu.RunMenu(g, items, handler)

	// Handle menu selection
	switch handler.GetSelectedAction() {
	case gamemenu.GameplayMenuActionBindings:
		// Open bindings menu (from gameplay menu, so show "Back" option)
		RunBindingsMenu(g, true) // true = from menu, shows "Back" option
		// After bindings menu closes, return to gameplay menu
		// (User can press F10 again to reopen gameplay menu)
	case gamemenu.GameplayMenuActionQuitToTitle:
		// Signal to quit to title - this will be handled by the game loop
		// We'll set a flag on the game state
		g.QuitToTitle = true
	}
}

// RunBindingsMenu presents a simple bindings configuration menu using the generic menu system.
// If fromMainMenu is true, a "Back" option will be available to return to the main menu.
func RunBindingsMenu(g *state.Game, fromMainMenu bool) {
	handler := gamemenu.NewBindingsMenuHandler(fromMainMenu)
	items := handler.GetMenuItems()
	gamemenu.RunMenu(g, items, handler)
}

// RunMaintenanceMenu shows the maintenance terminal menu with room devices and power consumption using the generic menu system.
func RunMaintenanceMenu(g *state.Game, cell *world.Cell, maintenanceTerm *entities.MaintenanceTerminal) {
	handler := gamemenu.NewMaintenanceMenuHandler(g, cell, maintenanceTerm)
	gamemenu.RunMenuDynamic(g, handler)
}
