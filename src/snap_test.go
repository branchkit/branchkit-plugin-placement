package main

import (
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

func makeTestWorld() (*branchkit.WindowInfo, branchkit.DisplayInfo, []branchkit.DisplayInfo) {
	win := &branchkit.WindowInfo{
		ID: "test-win", AppID: "com.test", AppName: "Test", Title: "Test",
		X: 100, Y: 100, W: 800, H: 600,
	}
	// A 1920x1080 display whose top 25 px is the menu bar: the usable area
	// the OS reports starts below it.
	display := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1920, H: 1080,
		VisibleX: 0, VisibleY: 25, VisibleW: 1920, VisibleH: 1055}
	return win, display, []branchkit.DisplayInfo{display}
}

func TestSnapLeft(t *testing.T) {
	win, screen, displays := makeTestWorld()
	r := calculateSnapGeometry(win, screen, 0, displays, "left")
	if r == nil {
		t.Fatal("expected geometry")
	}
	if r.X != 0 || r.Y != 25 || r.W != 960 || r.H != 1055 {
		t.Errorf("left: got (%d,%d %dx%d), want (0,25 960x1055)", r.X, r.Y, r.W, r.H)
	}
}

func TestSnapRight(t *testing.T) {
	win, screen, displays := makeTestWorld()
	r := calculateSnapGeometry(win, screen, 0, displays, "right")
	if r == nil {
		t.Fatal("expected geometry")
	}
	if r.X != 960 || r.Y != 25 || r.W != 960 || r.H != 1055 {
		t.Errorf("right: got (%d,%d %dx%d), want (960,25 960x1055)", r.X, r.Y, r.W, r.H)
	}
}

func TestSnapTop(t *testing.T) {
	win, screen, displays := makeTestWorld()
	r := calculateSnapGeometry(win, screen, 0, displays, "top")
	if r == nil {
		t.Fatal("expected geometry")
	}
	if r.X != 0 || r.Y != 25 || r.W != 1920 || r.H != 527 {
		t.Errorf("top: got (%d,%d %dx%d), want (0,25 1920x527)", r.X, r.Y, r.W, r.H)
	}
}

func TestSnapBottom(t *testing.T) {
	win, screen, displays := makeTestWorld()
	r := calculateSnapGeometry(win, screen, 0, displays, "bottom")
	if r == nil {
		t.Fatal("expected geometry")
	}
	// top half is 1055/2 = 527; the bottom half takes the other 528 at y = 25 + 527
	if r.X != 0 || r.Y != 552 || r.W != 1920 || r.H != 528 {
		t.Errorf("bottom: got (%d,%d %dx%d), want (0,552 1920x528)", r.X, r.Y, r.W, r.H)
	}
}

func TestSnapMaximize(t *testing.T) {
	win, screen, displays := makeTestWorld()
	for _, dir := range []string{"maximize", "full"} {
		r := calculateSnapGeometry(win, screen, 0, displays, dir)
		if r == nil {
			t.Fatalf("%s: expected geometry", dir)
		}
		if r.X != 0 || r.Y != 25 || r.W != 1920 || r.H != 1055 {
			t.Errorf("%s: got (%d,%d %dx%d), want (0,25 1920x1055)", dir, r.X, r.Y, r.W, r.H)
		}
	}
}

func TestSnapCenter(t *testing.T) {
	win, screen, displays := makeTestWorld()
	r := calculateSnapGeometry(win, screen, 0, displays, "center")
	if r == nil {
		t.Fatal("expected geometry")
	}
	if r.X != 480 || r.Y != 288 || r.W != 960 || r.H != 527 {
		t.Errorf("center: got (%d,%d %dx%d), want (480,288 960x527)", r.X, r.Y, r.W, r.H)
	}
}

func TestSnapNextMonitor(t *testing.T) {
	win := &branchkit.WindowInfo{
		ID: "test-win", AppID: "com.test", AppName: "Test", Title: "Test",
		X: 100, Y: 100, W: 800, H: 600,
	}
	d1 := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1920, H: 1080}
	d2 := branchkit.DisplayInfo{ID: 2, X: 1920, Y: 0, W: 2560, H: 1440}
	displays := []branchkit.DisplayInfo{d1, d2}

	r := calculateSnapGeometry(win, d1, 0, displays, "next")
	if r == nil {
		t.Fatal("expected geometry")
	}
	// Proportional: relX=100/1920, relY=100/1080, relW=800/1920, relH=600/1080
	// targetX = 1920 + round(100/1920*2560) = 1920+133 = 2053
	// targetY = 0 + round(100/1080*1440) = 133
	// targetW = round(800/1920*2560) = 1067
	// targetH = round(600/1080*1440) = 800
	if r.X != 2053 || r.Y != 133 || r.W != 1067 || r.H != 800 {
		t.Errorf("next: got (%d,%d %dx%d), want (2053,133 1067x800)", r.X, r.Y, r.W, r.H)
	}
}

func TestSnapPrevMonitor(t *testing.T) {
	win := &branchkit.WindowInfo{
		ID: "test-win", AppID: "com.test", AppName: "Test", Title: "Test",
		X: 2020, Y: 100, W: 1280, H: 720,
	}
	d1 := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1920, H: 1080}
	d2 := branchkit.DisplayInfo{ID: 2, X: 1920, Y: 0, W: 2560, H: 1440}
	displays := []branchkit.DisplayInfo{d1, d2}

	r := calculateSnapGeometry(win, d2, 1, displays, "prev")
	if r == nil {
		t.Fatal("expected geometry")
	}
	// Window at (2020,100) on d2(1920..4480). relX = (2020-1920)/2560 = 100/2560
	// targetX = 0 + round(100/2560*1920) = 75
	if r.X != 75 {
		t.Errorf("prev: X=%d, want 75", r.X)
	}
}

func TestSnapNextSingleMonitorReturnsNil(t *testing.T) {
	win, screen, displays := makeTestWorld()
	r := calculateSnapGeometry(win, screen, 0, displays, "next")
	if r != nil {
		t.Error("next on single monitor should return nil")
	}
}

func TestSnapUnknownReturnsNil(t *testing.T) {
	win, screen, displays := makeTestWorld()
	r := calculateSnapGeometry(win, screen, 0, displays, "bogus")
	if r != nil {
		t.Error("unknown direction should return nil")
	}
}

// The snapped event carries what a subscriber needs to let go of the window
// and know where it is going.
func TestSnappedEvent_Payload(t *testing.T) {
	got := string(snappedEvent("42", "left", branchkit.Rect{X: 0, Y: 25, W: 800, H: 875}))
	want := `{"frame":{"h":875,"w":800,"x":0,"y":25},"position":"left","window_id":"42"}`
	if got != want {
		t.Fatalf("payload = %s, want %s", got, want)
	}
}

func TestMovedToSpaceEvent_Payload(t *testing.T) {
	got := string(movedToSpaceEvent("42", 3, true))
	if want := `{"space":3,"stay":true,"window_id":"42"}`; got != want {
		t.Fatalf("payload = %s, want %s", got, want)
	}
}

// A Dock on the left and a taller menu bar (a notched MacBook): every
// position stays inside the usable area, never under either.
func TestSnapStaysInsideTheUsableArea(t *testing.T) {
	win, _, _ := makeTestWorld()
	d := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1512, H: 982,
		VisibleX: 64, VisibleY: 38, VisibleW: 1448, VisibleH: 944}
	for _, dir := range []string{"left", "right", "top", "bottom", "maximize", "center"} {
		r := calculateSnapGeometry(win, d, 0, []branchkit.DisplayInfo{d}, dir)
		if r == nil {
			t.Fatalf("%s: expected geometry", dir)
		}
		if r.X < 64 || r.Y < 38 || r.X+r.W > 64+1448 || r.Y+r.H > 38+944 {
			t.Errorf("%s: (%d,%d %dx%d) leaves the usable area (64,38 1448x944)", dir, r.X, r.Y, r.W, r.H)
		}
	}
	l := calculateSnapGeometry(win, d, 0, []branchkit.DisplayInfo{d}, "left")
	r := calculateSnapGeometry(win, d, 0, []branchkit.DisplayInfo{d}, "right")
	if l.X != 64 || l.X+l.W != r.X || r.X+r.W != 64+1448 {
		t.Errorf("halves must tile the usable area: left (%d,%d), right (%d,%d)", l.X, l.W, r.X, r.W)
	}
}

// A display that reports no visible bounds falls back to its full frame.
func TestSnapWithoutVisibleBoundsUsesTheFullDisplay(t *testing.T) {
	win, _, _ := makeTestWorld()
	d := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1920, H: 1080}
	r := calculateSnapGeometry(win, d, 0, []branchkit.DisplayInfo{d}, "maximize")
	if r.X != 0 || r.Y != 0 || r.W != 1920 || r.H != 1080 {
		t.Errorf("maximize: got (%d,%d %dx%d), want (0,0 1920x1080)", r.X, r.Y, r.W, r.H)
	}
}

// A window filling one display's usable area fills the next display's,
// even when the two areas are inset differently.
func TestSnapNextMonitorMapsBetweenUsableAreas(t *testing.T) {
	d1 := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1920, H: 1080,
		VisibleX: 0, VisibleY: 25, VisibleW: 1920, VisibleH: 1055}
	d2 := branchkit.DisplayInfo{ID: 2, X: 1920, Y: 0, W: 2560, H: 1440,
		VisibleX: 1920, VisibleY: 0, VisibleW: 2560, VisibleH: 1392} // taskbar-style bar at the bottom
	win := &branchkit.WindowInfo{ID: "w", X: 0, Y: 25, W: 1920, H: 1055}
	r := calculateSnapGeometry(win, d1, 0, []branchkit.DisplayInfo{d1, d2}, "next")
	if r == nil {
		t.Fatal("expected geometry")
	}
	if r.X != 1920 || r.Y != 0 || r.W != 2560 || r.H != 1392 {
		t.Errorf("next: got (%d,%d %dx%d), want (1920,0 2560x1392)", r.X, r.Y, r.W, r.H)
	}
}
