package ebiten

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"darkstation/pkg/game/config"
)

// monitorInfo is a portable description of a display for preference matching.
type monitorInfo struct {
	Name   string
	Width  int
	Height int
	Index  int
}

func listMonitorInfos() []monitorInfo {
	monitors := ebiten.AppendMonitors(nil)
	out := make([]monitorInfo, 0, len(monitors))
	for i, m := range monitors {
		if m == nil {
			continue
		}
		w, h := m.Size()
		out = append(out, monitorInfo{
			Name:   m.Name(),
			Width:  w,
			Height: h,
			Index:  i,
		})
	}
	return out
}

// selectPreferredMonitorIndex picks the best matching monitor for a saved preference.
// Returns -1 when nothing suitable is available.
func selectPreferredMonitorIndex(monitors []monitorInfo, prefName string, prefIndex, prefW, prefH int) int {
	if len(monitors) == 0 {
		return -1
	}
	if prefName == "" && prefIndex < 0 {
		return -1
	}

	nameMatches := make([]int, 0, len(monitors))
	for _, m := range monitors {
		if prefName != "" && m.Name == prefName {
			nameMatches = append(nameMatches, m.Index)
		}
	}

	// Prefer name + size (helps when several monitors share a generic name).
	if prefName != "" && prefW > 0 && prefH > 0 {
		for _, m := range monitors {
			if m.Name == prefName && m.Width == prefW && m.Height == prefH {
				return m.Index
			}
		}
	}

	// Unique name match.
	if len(nameMatches) == 1 {
		return nameMatches[0]
	}

	// Ambiguous names: use saved index when it still points at that name.
	if len(nameMatches) > 1 && prefIndex >= 0 {
		for _, idx := range nameMatches {
			if idx == prefIndex {
				return idx
			}
		}
		return nameMatches[0]
	}

	// Name missing (unplugged/renamed): try index + size, then index alone.
	if prefIndex >= 0 && prefIndex < len(monitors) {
		m := monitors[prefIndex]
		if prefW > 0 && prefH > 0 {
			if m.Width == prefW && m.Height == prefH {
				return prefIndex
			}
		} else {
			return prefIndex
		}
	}

	return -1
}

func restorePreferredMonitor(cfg *config.Config) {
	if cfg == nil {
		return
	}
	monitors := ebiten.AppendMonitors(nil)
	infos := listMonitorInfos()
	idx := selectPreferredMonitorIndex(infos, cfg.MonitorName, cfg.MonitorIndex, cfg.MonitorWidth, cfg.MonitorHeight)
	if idx < 0 || idx >= len(monitors) || monitors[idx] == nil {
		return
	}
	ebiten.SetMonitor(monitors[idx])
	log.Printf("Display: restored monitor %q (index %d, %dx%d)", monitors[idx].Name(), idx, cfg.MonitorWidth, cfg.MonitorHeight)
}

func persistCurrentMonitor(cfg *config.Config) {
	if cfg == nil {
		return
	}
	m := ebiten.Monitor()
	if m == nil {
		return
	}
	name := m.Name()
	w, h := m.Size()
	idx := -1
	for i, candidate := range ebiten.AppendMonitors(nil) {
		if candidate == m {
			idx = i
			break
		}
	}
	// Pointer equality can fail across calls; fall back to name+size match.
	if idx < 0 {
		for i, candidate := range ebiten.AppendMonitors(nil) {
			if candidate == nil {
				continue
			}
			cw, ch := candidate.Size()
			if candidate.Name() == name && cw == w && ch == h {
				idx = i
				break
			}
		}
	}
	if cfg.MonitorName == name && cfg.MonitorIndex == idx && cfg.MonitorWidth == w && cfg.MonitorHeight == h {
		return
	}
	if err := cfg.SetMonitorPreference(name, idx, w, h); err != nil {
		log.Printf("Display: could not save monitor preference: %v", err)
	}
}

// SaveMonitorPreference writes the current monitor into settings (best-effort).
func SaveMonitorPreference() {
	persistCurrentMonitor(config.Current())
}
