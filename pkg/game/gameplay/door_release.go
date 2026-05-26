package gameplay

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// DoorNeedsManualRelease reports whether the player can hold USE to mechanically release this door.
func DoorNeedsManualRelease(g *state.Game, cell *world.Cell) bool {
	return doorNeedsManualRelease(g, cell)
}

func doorNeedsManualRelease(g *state.Game, cell *world.Cell) bool {
	if g == nil || cell == nil || !gameworld.HasDoor(cell) {
		return false
	}
	door := gameworld.GetGameData(cell).Door
	if door == nil || door.RoomName == "" {
		return false
	}
	if setup.CellHasLivePower(g, cell) {
		return false
	}
	if g.ManualEgressReleased != nil && g.ManualEgressReleased[door.RoomName] {
		return false
	}
	return true
}

func manualEgressReleased(g *state.Game, roomName string) bool {
	return g != nil && g.ManualEgressReleased != nil && g.ManualEgressReleased[roomName]
}

func showManualDoorReleaseCallout(g *state.Game, cell *world.Cell) {
	if cell == nil || !gameworld.HasDoor(cell) {
		return
	}
	door := gameworld.GetGameData(cell).Door
	if door == nil {
		return
	}
	msg := fmt.Sprintf("UNPOWERED{%s}\nSUBTLE{Status: }UNPOWERED{No routing power}\nSUBTLE{Hold USE — manual egress release}", door.DoorName())
	renderer.AddCallout(cell.Row, cell.Col, msg, renderer.CalloutColorDoor, 0)
}

func completeManualDoorRelease(g *state.Game, cell *world.Cell) {
	if cell == nil || !gameworld.HasDoor(cell) {
		return
	}
	door := gameworld.GetGameData(cell).Door
	if door == nil || door.RoomName == "" {
		return
	}
	if g.ManualEgressReleased == nil {
		g.ManualEgressReleased = make(map[string]bool)
	}
	g.ManualEgressReleased[door.RoomName] = true
	renderer.AddCallout(cell.Row, cell.Col,
		"EGRESS{Manual release}\nSUBTLE{Door unsecured — routing still offline}",
		renderer.CalloutColorDoor, 0)
	logMessage(g, "Manual egress release: %s", door.DoorName())
	UpdateLightingExploration(g)
}
