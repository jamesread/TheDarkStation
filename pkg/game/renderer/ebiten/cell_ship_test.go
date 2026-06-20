package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/generator"
)

func TestOverlayRoomFloorRenderOptions_shipDistinct(t *testing.T) {
	shipCell := &world.Cell{Name: generator.ShipRoomName, Room: true}
	roomCell := &world.Cell{Name: "Crew Quarters", Room: true}

	shipOpts, ok := overlayRoomFloorRenderOptions(shipCell, false)
	if !ok {
		t.Fatal("ship overlay options missing")
	}
	if shipOpts.Icon != "⬡" {
		t.Fatalf("unvisited ship icon = %q, want ⬡", shipOpts.Icon)
	}
	if shipOpts.Color != colorShipFloor {
		t.Fatalf("ship color = %v, want %v", shipOpts.Color, colorShipFloor)
	}
	shipVisited, ok := overlayRoomFloorRenderOptions(shipCell, true)
	if !ok || shipVisited.Icon != "⬢" {
		t.Fatalf("visited ship icon = %q, want ⬢", shipVisited.Icon)
	}

	if _, ok := overlayRoomFloorRenderOptions(roomCell, false); ok {
		t.Fatal("regular room should not use overlay styling")
	}
}

func TestShipWallRenderOptions_adjacentToShip(t *testing.T) {
	wall := &world.Cell{Room: false, Discovered: true}
	ship := &world.Cell{Name: generator.ShipRoomName, Room: true, East: wall, West: &world.Cell{Room: true, Name: "Corridor"}}
	wall.West = ship

	opts, ok := shipWallRenderOptions(wall)
	if !ok {
		t.Fatal("wall adjacent to ship should use hull styling")
	}
	if opts.Icon != IconShipHullWall {
		t.Fatalf("ship wall icon = %q, want %q", opts.Icon, IconShipHullWall)
	}
	if opts.Color != colorShipWall {
		t.Fatalf("ship wall color = %v, want %v", opts.Color, colorShipWall)
	}
	if opts.BackgroundColor != colorShipWallBg {
		t.Fatalf("ship wall bg = %v, want %v", opts.BackgroundColor, colorShipWallBg)
	}

	otherWall := &world.Cell{Room: false, Discovered: true}
	otherRoom := &world.Cell{Name: "Crew Quarters", Room: true, East: otherWall}
	otherWall.West = otherRoom
	if _, ok := shipWallRenderOptions(otherWall); ok {
		t.Fatal("non-ship wall should not use hull styling")
	}
}
