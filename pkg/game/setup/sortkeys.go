package setup

import (
	"sort"

	"darkstation/pkg/engine/world"
)

// SortCellsByPosition orders cells by row then column for deterministic selection.
func SortCellsByPosition(cells []*world.Cell) {
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Row != cells[j].Row {
			return cells[i].Row < cells[j].Row
		}
		return cells[i].Col < cells[j].Col
	})
}

func sortedRoomNames[V any](m map[string]V) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
