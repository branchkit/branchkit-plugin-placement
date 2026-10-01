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
	display := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1920, H: 1080}
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
	// halfH = (1080-25)/2 = 527, y = 25 + 527 = 552
	if r.X != 0 || r.Y != 552 || r.W != 1920 || r.H != 527 {
		t.Errorf("bottom: got (%d,%d %dx%d), want (0,552 1920x527)", r.X, r.Y, r.W, r.H)
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
	if r.X != 480 || r.Y != 270 || r.W != 960 || r.H != 540 {
		t.Errorf("center: got (%d,%d %dx%d), want (480,270 960x540)", r.X, r.Y, r.W, r.H)
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
