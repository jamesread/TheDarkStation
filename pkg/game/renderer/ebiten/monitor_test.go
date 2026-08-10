package ebiten

import "testing"

func TestSelectPreferredMonitorIndex(t *testing.T) {
	monitors := []monitorInfo{
		{Name: "HDMI-1", Width: 1920, Height: 1080, Index: 0},
		{Name: "DP-1", Width: 2560, Height: 1440, Index: 1},
		{Name: "Generic PnP Monitor", Width: 1920, Height: 1080, Index: 2},
		{Name: "Generic PnP Monitor", Width: 3840, Height: 2160, Index: 3},
	}

	cases := []struct {
		name                    string
		prefName                string
		prefIndex, prefW, prefH int
		want                    int
	}{
		{"empty pref", "", -1, 0, 0, -1},
		{"unique name", "DP-1", -1, 0, 0, 1},
		{"name and size", "Generic PnP Monitor", 0, 3840, 2160, 3},
		{"ambiguous name uses index", "Generic PnP Monitor", 2, 0, 0, 2},
		{"missing name falls back to index+size", "Gone", 1, 2560, 1440, 1},
		{"missing entirely", "Gone", 9, 800, 600, -1},
		{"index alone when no size saved", "Gone", 0, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := selectPreferredMonitorIndex(monitors, tc.prefName, tc.prefIndex, tc.prefW, tc.prefH)
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
