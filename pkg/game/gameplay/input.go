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

	case engineinput.ActionOpenInventory:
		gamemenu.RunInventoryMenu(g)
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
		abandonCouplerCrankOnMove(g)
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.East)
		return

	case engineinput.ActionMoveWest:
		abandonCouplerCrankOnMove(g)
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.West)
		return

	case engineinput.ActionMoveNorth:
		abandonCouplerCrankOnMove(g)
		g.NavStyle = state.NavStyleNSEW
		if g.CurrentCell == nil {
			return
		}
		MoveCell(g, g.CurrentCell.North)
		return

	case engineinput.ActionMoveSouth:
		abandonCouplerCrankOnMove(g)
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
	case gamemenu.GameplayMenuActionSettings:
		RunSettingsMenu(g, false)
	case gamemenu.GameplayMenuActionQuitToTitle:
		QuitToTitleMenu(g)
	}
}

// RunSettingsMenu opens the unified settings menu (bindings and video tabs).
// If fromMainMenu is true, Back returns to the title screen.
func RunSettingsMenu(g *state.Game, fromMainMenu bool) {
	handler := gamemenu.NewSettingsMenuHandler(fromMainMenu)
	gamemenu.RunMenuDynamic(g, handler)
}

// RunBindingsMenu opens settings on the bindings tab (legacy entry point).
func RunBindingsMenu(g *state.Game, fromMainMenu bool) {
	handler := gamemenu.NewSettingsMenuHandlerWithTab(fromMainMenu, gamemenu.SettingsTabBindings)
	gamemenu.RunMenuDynamic(g, handler)
}

// RunVideoMenu opens settings on the video tab (legacy entry point).
func RunVideoMenu(g *state.Game) {
	handler := gamemenu.NewSettingsMenuHandlerWithTab(false, gamemenu.SettingsTabVideo)
	gamemenu.RunMenuDynamic(g, handler)
}

// RunMaintenanceMenu shows the maintenance terminal menu with room devices and power consumption using the generic menu system.
func RunMaintenanceMenu(g *state.Game, cell *world.Cell, maintenanceTerm *entities.MaintenanceTerminal) {
	handler := gamemenu.NewMaintenanceMenuHandler(g, cell, maintenanceTerm)
	gamemenu.RunMenuDynamic(g, handler)
}

// RunRoutingCouplerMenu opens the lift routing coupler alignment mini-game.
func RunRoutingCouplerMenu(g *state.Game, cell *world.Cell, repair *entities.RepairObjective) {
	if g == nil || repair == nil {
		return
	}
	handler := gamemenu.NewRoutingCouplerMenuHandler(g, cell, repair, func() {
		completeRepair(g, repair, cell)
	})
	gamemenu.RunMenuDynamic(g, handler)
}
