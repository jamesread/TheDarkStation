package gameplay

import (
	"fmt"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const obsPlaquePrefix = "ENV_PLAQUE_OBS_"

// maybeAnnounceObservationCueOnMove shows one thin technical callout the first time
// the player enters a corridor cell whose plaque was retargeted for Story 5.2.
func maybeAnnounceObservationCueOnMove(g *state.Game, cell *world.Cell) {
	if g == nil || cell == nil {
		return
	}
	msg := gameworld.GetGameData(cell).EnvPlaqueMsgID
	if msg == "" || !strings.HasPrefix(msg, obsPlaquePrefix) {
		return
	}
	if g.ObservationCueVisited == nil {
		g.ObservationCueVisited = make(map[string]struct{})
	}
	key := fmt.Sprintf("%d:%d:%d", g.CurrentDeckID, cell.Row, cell.Col)
	if _, ok := g.ObservationCueVisited[key]; ok {
		return
	}
	g.ObservationCueVisited[key] = struct{}{}

	renderer.AddCallout(
		cell.Row, cell.Col,
		"TITLE{Structural stamp}\nSUBTLE{Notation may correlate with unsecured access datum lines.}",
		renderer.CalloutColorInfo,
		4000,
	)
}
