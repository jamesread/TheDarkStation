package gameplay

import (
	"fmt"
	"strings"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

const linkPlaquePrefix = "ENV_PLAQUE_LINK_"

// maybeAnnounceLinkageCueOnMove shows a thin callout the first time the player enters a linkage junction (Story 5.3).
func maybeAnnounceLinkageCueOnMove(g *state.Game, cell *world.Cell) {
	if g == nil || cell == nil {
		return
	}
	msg := gameworld.GetGameData(cell).EnvPlaqueMsgID
	if msg == "" || !strings.HasPrefix(msg, linkPlaquePrefix) {
		return
	}
	if g.LinkageCueVisited == nil {
		g.LinkageCueVisited = make(map[string]struct{})
	}
	key := fmt.Sprintf("%d:%d:%d", g.CurrentDeckID, cell.Row, cell.Col)
	if _, ok := g.LinkageCueVisited[key]; ok {
		return
	}
	g.LinkageCueVisited[key] = struct{}{}

	renderer.AddCallout(
		cell.Row, cell.Col,
		"TITLE{Relay stencil}\nSUBTLE{Correlate this label with access logs or furniture tags.}",
		renderer.CalloutColorInfo,
		4000,
	)
}
