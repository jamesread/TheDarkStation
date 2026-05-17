package gameplay

import (
	"regexp"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

var relayLinePattern = regexp.MustCompile(`(?i)relay\s*:\s*([A-Za-z0-9-]+)`)

// noteLinkageRelaysFromText records tokens from "Relay: TOKEN" lines in furniture or diegetic text (Story 5.3).
func noteLinkageRelaysFromText(g *state.Game, text string) {
	if g == nil || text == "" {
		return
	}
	m := relayLinePattern.FindStringSubmatch(text)
	if len(m) < 2 {
		return
	}
	g.RecordLinkageToken(m[1])
}

// noteLinkageTagFromVisitedCell records linkage when the player steps on a cell stamped with LinkageTag.
func noteLinkageTagFromVisitedCell(g *state.Game, cell *world.Cell) {
	if g == nil || cell == nil {
		return
	}
	tag := gameworld.GetGameData(cell).LinkageTag
	if tag != "" {
		g.RecordLinkageToken(tag)
	}
}
