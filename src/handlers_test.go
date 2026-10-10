package main

import (
	"encoding/json"
	"errors"
	"runtime"
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// fakePlatform stands in for the platform so a test can see which window a
// handler acted on. It embeds the interface unset: a call the test did not
// expect panics instead of passing quietly.
type fakePlatform struct {
	platformCalls

	world   branchkit.WorldModel
	focused string                                          // NativeFocusedWindowID; "" = none
	bounds  map[string]branchkit.NativeWindowBoundsResponse // NativeWindowBounds by id

	boundsAsked []string
	emitted     []map[string]any
	warps       []branchkit.NativeWarpCursorRequest
	moved       []branchkit.NativeMoveWindowToSpaceRequest
	frames      []branchkit.WindowFrame
}

func (f *fakePlatform) NativeWorldModel(branchkit.NativeWorldModelRequest) (*branchkit.WorldModel, error) {
	w := f.world
	return &w, nil
}

func (f *fakePlatform) NativeFocusedWindowID() (*branchkit.NativeFocusedWindowIDResponse, error) {
	if f.focused == "" {
		return nil, errors.New("nothing focused")
	}
	return &branchkit.NativeFocusedWindowIDResponse{WindowID: f.focused}, nil
}

func (f *fakePlatform) NativeWindowBounds(req branchkit.NativeWindowBoundsRequest) (*branchkit.NativeWindowBoundsResponse, error) {
	f.boundsAsked = append(f.boundsAsked, req.WindowID)
	b, ok := f.bounds[req.WindowID]
	if !ok {
		return nil, errors.New("no such window")
	}
	return &b, nil
}

func (f *fakePlatform) EventsEmit(req branchkit.EventsEmitRequest) error {
	var data map[string]any
	if err := json.Unmarshal(req.Data, &data); err != nil {
		return err
	}
	f.emitted = append(f.emitted, data)
	return nil
}

// The macOS move: warp to the title bar, drag, switch desktop.
func (f *fakePlatform) NativeCursorInfo() (*branchkit.NativeCursorInfoResponse, error) {
	return nil, errors.New("no cursor")
}
func (f *fakePlatform) NativeWarpCursor(req branchkit.NativeWarpCursorRequest) error {
	f.warps = append(f.warps, req)
	return nil
}
func (f *fakePlatform) InputMouseButton(branchkit.InputMouseButtonRequest) error { return nil }
func (f *fakePlatform) NativeSwitchSpace(branchkit.NativeSwitchSpaceRequest) error {
	return nil
}

// The move everywhere else: the OS moves the window itself.
func (f *fakePlatform) NativeListSpaces() ([]branchkit.SpaceInfo, error) {
	return []branchkit.SpaceInfo{
		{SpaceID: 101, SpaceType: "user"},
		{SpaceID: 102, SpaceType: "user"},
	}, nil
}
func (f *fakePlatform) NativeMoveWindowToSpace(req branchkit.NativeMoveWindowToSpaceRequest) (bool, error) {
	f.moved = append(f.moved, req)
	return true, nil
}

func (f *fakePlatform) NativeBatchSetFrames(req branchkit.NativeBatchSetFramesRequest) ([]branchkit.WindowFrame, error) {
	f.frames = append(f.frames, req.Frames...)
	return nil, nil
}

func strPtr(s string) *string { return &s }

// assertMovedWindow checks that move_to_space moved window `id`, whose
// frame starts at (x, y): the event names it, and the move itself acts on
// it (macOS grabs its title bar; elsewhere the OS is asked to move it).
func assertMovedWindow(t *testing.T, f *fakePlatform, id string, x, y int) {
	t.Helper()
	if len(f.emitted) != 1 || f.emitted[0]["window_id"] != id {
		t.Fatalf("moved_to_space events = %v, want one naming %s", f.emitted, id)
	}
	if runtime.GOOS == "darwin" {
		if len(f.warps) == 0 || f.warps[0].X != x+75 || f.warps[0].Y != y+10 {
			t.Fatalf("warps = %v, want the first on %s's title bar (%d, %d)", f.warps, id, x+75, y+10)
		}
		return
	}
	if len(f.moved) != 1 || f.moved[0].WindowID != id || f.moved[0].SpaceID != 102 {
		t.Fatalf("moves = %v, want %s to space 102", f.moved, id)
	}
}

// A dispatching plugin (browser tab-to-desk) names the window it just
// created; that window, not the focused one, is the one that moves.
func TestMoveToSpaceExplicitWindowIDWinsOverTheActiveWindow(t *testing.T) {
	f := &fakePlatform{world: branchkit.WorldModel{
		ActiveWindowID: strPtr("focused"),
		Windows: []branchkit.WindowInfo{
			{ID: "focused", X: 0, Y: 0, W: 800, H: 600},
			{ID: "new", X: 500, Y: 300, W: 800, H: 600},
		},
	}}
	h := &Host{plugin: f}
	req := &branchkit.OnActionRequest{ActiveWindowID: strPtr("focused")}

	if _, err := h.handleWindowsMoveToSpace(MoveToSpaceParams{Space: "2", WindowID: strPtr("new")}, req); err != nil {
		t.Fatal(err)
	}
	assertMovedWindow(t, f, "new", 500, 300)
}

// With no window named, the envelope's active window moves.
func TestMoveToSpaceWithoutWindowIDMovesTheActiveWindow(t *testing.T) {
	f := &fakePlatform{world: branchkit.WorldModel{
		Windows: []branchkit.WindowInfo{
			{ID: "other", X: 0, Y: 0, W: 800, H: 600},
			{ID: "focused", X: 40, Y: 60, W: 800, H: 600},
		},
	}}
	h := &Host{plugin: f}
	req := &branchkit.OnActionRequest{ActiveWindowID: strPtr("focused")}

	if _, err := h.handleWindowsMoveToSpace(MoveToSpaceParams{Space: "2", WindowID: strPtr("")}, req); err != nil {
		t.Fatal(err)
	}
	assertMovedWindow(t, f, "focused", 40, 60)
}

// The world model can miss a window that just appeared: its bounds are
// asked for directly and the move goes ahead.
func TestMoveToSpaceFallsBackToWindowBoundsWhenTheWorldModelMisses(t *testing.T) {
	f := &fakePlatform{
		world:  branchkit.WorldModel{Windows: []branchkit.WindowInfo{{ID: "old", X: 0, Y: 0, W: 10, H: 10}}},
		bounds: map[string]branchkit.NativeWindowBoundsResponse{"new": {X: 200, Y: 120, W: 640, H: 480}},
	}
	h := &Host{plugin: f}

	if _, err := h.handleWindowsMoveToSpace(MoveToSpaceParams{Space: "2", WindowID: strPtr("new")}, &branchkit.OnActionRequest{}); err != nil {
		t.Fatal(err)
	}
	if len(f.boundsAsked) != 1 || f.boundsAsked[0] != "new" {
		t.Fatalf("bounds asked for %v, want [new]", f.boundsAsked)
	}
	assertMovedWindow(t, f, "new", 200, 120)
}

// No window named and none active in the envelope or the world model: the
// focused window is asked for, then its bounds.
func TestMoveToSpaceFallsBackToTheFocusedWindow(t *testing.T) {
	f := &fakePlatform{
		focused: "lone",
		bounds:  map[string]branchkit.NativeWindowBoundsResponse{"lone": {X: 10, Y: 20, W: 300, H: 200}},
	}
	h := &Host{plugin: f}

	if _, err := h.handleWindowsMoveToSpace(MoveToSpaceParams{Space: "2"}, &branchkit.OnActionRequest{}); err != nil {
		t.Fatal(err)
	}
	assertMovedWindow(t, f, "lone", 10, 20)
}

// Nothing found by any route is an error, and nothing is announced or moved.
func TestMoveToSpaceWithNoWindowIsAnError(t *testing.T) {
	f := &fakePlatform{}
	h := &Host{plugin: f}

	if _, err := h.handleWindowsMoveToSpace(MoveToSpaceParams{Space: "2", WindowID: strPtr("gone")}, &branchkit.OnActionRequest{}); err == nil {
		t.Fatal("want an error when the window is nowhere")
	}
	if len(f.emitted)+len(f.warps)+len(f.moved) != 0 {
		t.Fatalf("emitted %v, warped %v, moved %v: want nothing", f.emitted, f.warps, f.moved)
	}
}

// Snap acts on the envelope's active window when it names one, and on the
// world model's active window otherwise.
func TestSnapTargetsTheEnvelopeWindowThenTheWorldModels(t *testing.T) {
	display := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1000, H: 800}
	world := branchkit.WorldModel{
		ActiveWindowID: strPtr("wm-active"),
		Displays:       []branchkit.DisplayInfo{display},
		Windows: []branchkit.WindowInfo{
			{ID: "wm-active", X: 100, Y: 100, W: 400, H: 300},
			{ID: "envelope", X: 200, Y: 200, W: 400, H: 300},
		},
	}
	left := SnapPositionLeft

	for _, tc := range []struct {
		name   string
		active *string
		want   string
	}{
		{"envelope names a window", strPtr("envelope"), "envelope"},
		{"envelope names none", nil, "wm-active"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakePlatform{world: world}
			h := &Host{plugin: f, history: newFrameHistory()}
			req := &branchkit.OnActionRequest{ActiveWindowID: tc.active}
			if _, err := h.handleWindowsSnap(SnapParams{Position: &left}, req); err != nil {
				t.Fatal(err)
			}
			if len(f.frames) != 1 || f.frames[0].WindowID != tc.want {
				t.Fatalf("frames = %v, want one for %s", f.frames, tc.want)
			}
			if len(f.emitted) != 1 || f.emitted[0]["window_id"] != tc.want {
				t.Fatalf("snapped events = %v, want one naming %s", f.emitted, tc.want)
			}
		})
	}
}
