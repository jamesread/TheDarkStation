package menu

import (
	"fmt"
	"sort"
	"unicode"

	engworld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/setup"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"

	"github.com/leonelquinteros/gotext"
)

func functionalAbbrev(t deck.Type) string {
	switch t {
	case deck.Habitation:
		return "HAB"
	case deck.Research:
		return "RES"
	case deck.Logistics:
		return "LOG"
	case deck.PowerDistribution:
		return "PWR"
	case deck.EmergencySystems:
		return "EMG"
	case deck.CoreInfrastructure:
		return "COR"
	default:
		return "SYS"
	}
}

func roomNameSlug(room string, maxRunes int) string {
	var b []rune
	for _, r := range room {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b = append(b, r)
			if len(b) >= maxRunes {
				break
			}
		}
	}
	if len(b) == 0 {
		return "ROOM"
	}
	return string(b)
}

func strataSeed(g *state.Game, room string) uint32 {
	var h uint32 = 2166136261
	for _, ch := range room {
		h ^= uint32(ch)
		h *= 16777619
	}
	h ^= uint32(g.Level) * 0x9e3779b1
	h ^= uint32(g.CurrentDeckID) * 0x85ebca6b
	if g.LevelSeed != 0 {
		h ^= uint32(g.LevelSeed)
		h ^= uint32(g.LevelSeed >> 32)
	}
	return h
}

func collectLocalCorrelates(g *state.Game, selectedRoom string) []string {
	if g == nil || g.Grid == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var xcore, jnct, env []string

	addSeen := func(key, line string, bucket *[]string) {
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		*bucket = append(*bucket, line)
	}

	g.Grid.ForEachCell(func(row, col int, cell *engworld.Cell) {
		if cell == nil || !cell.Room || cell.Name != selectedRoom {
			return
		}
		d := gameworld.GetGameData(cell)
		if d.Puzzle != nil && d.Puzzle.LinkageToken != "" {
			key := "puz:" + d.Puzzle.LinkageToken
			line := fmt.Sprintf("XCORE-\trelay=%s\t%s", d.Puzzle.LinkageToken, gotext.Get("MAINT_DIAG_XCORE_TAIL"))
			addSeen(key, line, &xcore)
		}

		for _, nb := range cell.GetNeighbors() {
			if nb == nil || !setup.IsCorridorJunctionLayer(nb) {
				continue
			}
			gd := gameworld.GetGameData(nb)
			if gd.LinkageTag != "" {
				key := "lnk:" + gd.LinkageTag
				line := fmt.Sprintf("JNCT-\t%s\t%s", gd.LinkageTag, gotext.Get("MAINT_DIAG_JNCT_TAIL"))
				addSeen(key, line, &jnct)
			}
			if gd.EnvPlaqueMsgID != "" {
				key := "env:" + gd.EnvPlaqueMsgID
				line := fmt.Sprintf("ENVREF-\t%s\t%s", gd.EnvPlaqueMsgID, gotext.Get("MAINT_DIAG_ENVREF_TAIL"))
				addSeen(key, line, &env)
			}
		}
	})

	return mergeCorrelatesLimited(xcore, jnct, env, correlateLimitDefault)
}

const correlateLimitDefault = 6

// mergeCorrelatesLimited orders puzzle linkage lines before junction stamps before signage refs,
// sorts within each tier, then truncates — avoids alphabet sort hiding XCORE behind ENVREF (Story 5.4).
func mergeCorrelatesLimited(xcore, jnct, env []string, limit int) []string {
	sort.Strings(xcore)
	sort.Strings(jnct)
	sort.Strings(env)
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for _, tier := range [][]string{xcore, jnct, env} {
		for _, line := range tier {
			if len(out) >= limit {
				return out
			}
			out = append(out, line)
		}
	}
	return out
}

// maintenanceInstrumentMenuLines builds read-only diagnostic labels for the maintenance menu (Story 5.4).
func maintenanceInstrumentMenuLines(g *state.Game, selectedRoom string) []string {
	if g == nil || g.Grid == nil || selectedRoom == "" {
		return nil
	}

	seed := strataSeed(g, selectedRoom)
	ft := deck.FunctionalType(g.Level)
	abbr := functionalAbbrev(ft)
	slug := roomNameSlug(selectedRoom, 4)

	t1 := seed ^ 0xa5a5a5a5
	t2 := seed ^ 0xc3c3c3c3
	t3 := seed ^ 0x3c3c3c3c
	flt := (seed >> 8) & 0xfff

	lines := []string{
		"SUBTLE{" + gotext.Get("MAINT_DIAG_TRACE_HEADER") + "}",
		fmt.Sprintf("LOG\tT+%03X-%03X-%03X\t%s", t1&0xfff, t2&0xfff, t3&0xfff, gotext.Get("MAINT_DIAG_LOG_CLOCK_SUFFIX")),
		fmt.Sprintf("SUBSYS-\tBUS-%s-%s-%02X\t%s", abbr, slug, seed&0xff, gotext.Get("MAINT_DIAG_SUBSYS_SUFFIX")),
		fmt.Sprintf("FLT-\tFLT-%03X\t%s", flt, gotext.Get("MAINT_DIAG_FLT_ADVISORY")),
		fmt.Sprintf("CLK-\tdeck=%d level=%d\t%s", g.CurrentDeckID+1, g.Level, gotext.Get("MAINT_DIAG_CLK_SYNC")),
	}

	cors := collectLocalCorrelates(g, selectedRoom)
	if len(cors) == 0 {
		return lines
	}

	lines = append(lines, "")
	lines = append(lines, "SUBTLE{"+gotext.Get("MAINT_DIAG_CORRELATES_HEADER")+"}")
	lines = append(lines, cors...)
	return lines
}
