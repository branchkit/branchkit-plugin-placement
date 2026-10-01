package main

import (
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

func TestOverviewKeyIsEachOSOwn(t *testing.T) {
	for goos, want := range map[string]string{"darwin": "ctrl+up", "windows": "meta+tab", "linux": "meta", "freebsd": "meta"} {
		name, mods := overviewKey(goos)
		got := name
		if len(mods) > 0 {
			got = mods[0] + "+" + name
		}
		if got != want {
			t.Errorf("%s: %s, want %s", goos, got, want)
		}
	}
}

func win(id, app string, minimized bool) branchkit.WindowInfo {
	return branchkit.WindowInfo{ID: id, AppID: app, AppName: app, IsMinimized: &minimized}
}

// Saying it again walks every window of the app and comes back round;
// other apps' and minimized windows are skipped; ids sort as numbers.
func TestNextAppWindowCyclesTheFocusedAppsWindows(t *testing.T) {
	ws := []branchkit.WindowInfo{
		win("100", "editor", false),
		win("9", "editor", false),
		win("50", "browser", false),
		win("70", "editor", true),
		win("20", "editor", false),
	}
	cur := "9"
	var seen []string
	for i := 0; i < 3; i++ {
		next, err := nextAppWindow(ws, cur)
		if err != nil {
			t.Fatal(err)
		}
		seen = append(seen, next)
		cur = next
	}
	want := []string{"20", "100", "9"}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("cycle = %v, want %v", seen, want)
		}
	}
}

func TestNextAppWindowSaysWhyItCannot(t *testing.T) {
	if _, err := nextAppWindow([]branchkit.WindowInfo{win("1", "editor", false)}, "1"); err == nil {
		t.Error("one window: want an error")
	}
	if _, err := nextAppWindow([]branchkit.WindowInfo{win("1", "editor", false)}, ""); err == nil {
		t.Error("nothing focused: want an error")
	}
}
