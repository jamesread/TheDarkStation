// Package gameplay provides core game logic for player movement and interactions.
package gameplay

import (
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/leonelquinteros/gotext"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/devtools"
	"darkstation/pkg/game/entities"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// ProcessIntent handles a high-level input intent from the tiered input system.
func ProcessIntent(g *state.Game, intent engineinput.Intent) {
	if g.GameComplete {
		ProcessCompletionInput(g, intent)
		return
	}

	if IsGameplayCinematicActive(g) {
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
		if gamemenu.ConfirmQuitGame(g) {
			fmt.Println(gotext.Get("GOODBYE"))
			os.Exit(0)
		}

	case engineinput.ActionScreenshot:
		filename := devtools.SaveScreenshotHTML(g)
		logMessage(g, "Screenshot saved to ITEM{%s}", filename)
		return

	case engineinput.ActionDevMenu:
		RunDeveloperMenu(g)
		return

	case engineinput.ActionDevMap:
		devtools.SwitchToDevMap(g)
		return

	case engineinput.ActionMaintPanTestMap:
		devtools.SwitchToMaintPanTestMap(g)
		return

	case engineinput.ActionPerfTestMap:
		devtools.SwitchToPerfMap(g, intent.Code)
		return

	case engineinput.ActionDebugMapDump:
		path, err := devtools.DumpRevealedMapToFile(g)
		if err != nil {
			logMessage(g, "Map dump failed: %v", err)
		} else {
			renderer.ShowDeveloperMessage(renderer.FormatText("Map dumped to ITEM{%s}", path))
		}
		return

	case engineinput.ActionResetLevel:
		ResetLevel(g)
		return

	case engineinput.ActionMoveEast:
		CancelLongUse(g)
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.East)
		return

	case engineinput.ActionMoveWest:
		CancelLongUse(g)
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.West)
		return

	case engineinput.ActionMoveNorth:
		CancelLongUse(g)
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.North)
		return

	case engineinput.ActionMoveSouth:
		CancelLongUse(g)
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.South)
		return

	case engineinput.ActionInteract:
		log.Printf("[Interact] ProcessIntent: ActionInteract (game loop tick)")
		if cell, kind, ok := findAdjacentLongUseTarget(g); ok {
			FaceTowardAdjacentCell(g, cell)
			switch kind {
			case LongUseGeneratorPowerUp:
				CheckAdjacentGeneratorAtCell(g, cell)
			case LongUseDoorManualRelease:
				showManualDoorReleaseCallout(g, cell)
			case LongUseRepair:
				if gameworld.HasRepairDevice(cell) {
					repair := gameworld.GetGameData(cell).RepairDevice
					renderer.AddCallout(cell.Row, cell.Col, repairDeviceCallout(g, repair, cell), renderer.CalloutColorMaintenance, 0)
				}
			}
		}
		if TryBeginLongUseOnAdjacent(g) {
			log.Printf("[Interact] ProcessIntent: started long-use hold interaction")
			return
		}
		if g.CurrentCell != nil && g.CurrentCell.ExitCell {
			if TryUseLift(g) {
				return
			}
		}
		interacted := CheckAdjacentInteractables(g)
		log.Printf("[Interact] ProcessIntent: CheckAdjacentInteractables returned %v", interacted)
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
	case gamemenu.GameplayMenuActionInventory:
		gamemenu.RunInventoryMenu(g)
	case gamemenu.GameplayMenuActionBindings:
		// Open bindings menu (from gameplay menu, so show "Back" option)
		RunBindingsMenu(g, true) // true = from menu, shows "Back" option
		// After bindings menu closes, return to gameplay menu
		// (User can press F10 again to reopen gameplay menu)
	case gamemenu.GameplayMenuActionVideo:
		RunVideoMenu(g)
	case gamemenu.GameplayMenuActionQuitToTitle:
		QuitToTitleMenu(g)
	}
}

// RunBindingsMenu presents a simple bindings configuration menu using the generic menu system.
// If fromMainMenu is true, a "Back" option will be available to return to the main menu.
func RunBindingsMenu(g *state.Game, fromMainMenu bool) {
	handler := gamemenu.NewBindingsMenuHandler(fromMainMenu)
	items := handler.GetMenuItems()
	gamemenu.RunMenu(g, items, handler)
}

// RunVideoMenu presents display settings.
func RunVideoMenu(g *state.Game) {
	handler := gamemenu.NewVideoMenuHandler()
	gamemenu.RunMenuDynamic(g, handler)
}

// RunMaintenanceMenu shows the maintenance terminal menu with room devices and power consumption using the generic menu system.
func RunMaintenanceMenu(g *state.Game, cell *world.Cell, maintenanceTerm *entities.MaintenanceTerminal) {
	handler := gamemenu.NewMaintenanceMenuHandler(g, cell, maintenanceTerm)
	gamemenu.RunMenuDynamic(g, handler)
}
