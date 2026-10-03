package main

import (
	"strings"
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// Three screens as they sit on a desk: a laptop at the bottom left, a
// big monitor beside it, and one mounted above the monitor. The OS lists
// them in no particular order.
func deskScreens() []branchkit.DisplayInfo {
	return []branchkit.DisplayInfo{
		{ID: 10, X: 1512, Y: 0, W: 2560, H: 1440},     // monitor
		{ID: 20, X: 1512, Y: -1440, W: 2560, H: 1440}, // above the monitor
		{ID: 30, X: 0, Y: 458, W: 1512, H: 982},       // laptop, left
	}
}

func TestScreenNumbersCountLeftToRight(t *testing.T) {
	d := deskScreens()
	for n, wantID := range map[string]int{"1": 30, "2": 20, "3": 10} {
		i, err := targetScreen(d, 0, n)
		if err != nil || d[i].ID != wantID {
			t.Errorf("screen %s: got index %d (%v), want display %d", n, i, err, wantID)
		}
	}
	if _, err := targetScreen(d, 0, "4"); err == nil || !strings.Contains(err.Error(), "has 3") {
		t.Errorf("screen 4: err = %v", err)
	}
}

func TestScreenDirectionsPickTheNearestScreenThatWay(t *testing.T) {
	d := deskScreens()
	cases := []struct {
		from   int
		dir    string
		wantID int
	}{
		{0, "left", 30}, {0, "up", 20}, {2, "right", 10}, {1, "down", 10},
	}
	for _, c := range cases {
		i, err := targetScreen(d, c.from, c.dir)
		if err != nil || d[i].ID != c.wantID {
			t.Errorf("from %d %s: got index %d (%v), want display %d", d[c.from].ID, c.dir, i, err, c.wantID)
		}
	}
	if _, err := targetScreen(d, 2, "left"); err == nil || !strings.Contains(err.Error(), "no screen to the left") {
		t.Errorf("nothing left of the laptop: err = %v", err)
	}
}
