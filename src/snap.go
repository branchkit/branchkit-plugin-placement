package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/branchkit/plugin-sdk-go"
)

// snappedEventType is emitted for every snap, before the window moves.
const snappedEventType = "placement.snapped"

// snappedEvent is the payload: which window, the position asked for, and
// the frame it is about to get.
func snappedEvent(windowID, position string, frame branchkit.Rect) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"window_id": windowID,
		"position":  position,
		"frame":     map[string]int{"x": frame.X, "y": frame.Y, "w": frame.W, "h": frame.H},
	})
	return b
}

// handleSnap calculates snap geometry and applies it via batch-set-frames.
// The error says what kept the window from moving, for the caller to show.
func (h *Host) handleSnap(activeWindowID *string, direction string) error {
	start := time.Now()
	wm, err := h.plugin.NativeWorldModel(branchkit.NativeWorldModelRequest{})
	if err != nil {
		return fmt.Errorf("snap: read the windows: %w", err)
	}

	winID := ""
	if activeWindowID != nil {
		winID = *activeWindowID
	} else if wm.ActiveWindowID != nil {
		winID = *wm.ActiveWindowID
	}
	if winID == "" {
		return fmt.Errorf("snap: no window is focused")
	}

	var win *branchkit.WindowInfo
	for i := range wm.Windows {
		if wm.Windows[i].ID == winID {
			win = &wm.Windows[i]
			break
		}
	}
	if win == nil {
		return fmt.Errorf("snap: window %s is not on screen", winID)
	}

	if len(wm.Displays) == 0 {
		return fmt.Errorf("snap: no display is connected")
	}

	// Find which display the window center is on
	centerX := win.X + win.W/2
	centerY := win.Y + win.H/2
	screenIdx := 0
	for i, d := range wm.Displays {
		if centerX >= d.X && centerX < d.X+d.W && centerY >= d.Y && centerY < d.Y+d.H {
			screenIdx = i
			break
		}
	}
	screen := wm.Displays[screenIdx]

	frame := calculateSnapGeometry(win, screen, screenIdx, wm.Displays, direction)
	if frame == nil {
		if (direction == "next" || direction == "prev") && len(wm.Displays) < 2 {
			return fmt.Errorf("snap %s: there is only one display", direction)
		}
		return fmt.Errorf("snap: %q is not a position", direction)
	}

	branchkit.Logf("placement", "snap: window=%s direction=%s → x=%d y=%d w=%d h=%d (screen %d: %dx%d)",
		winID, direction, frame.X, frame.Y, frame.W, frame.H,
		screenIdx, screen.W, screen.H)

	// Say so BEFORE moving it. A plugin managing this window (a tiler)
	// takes it as "the user placed this one by hand" and lets go of it;
	// told after the move, it may already have seen the window leave its
	// slot, read that as a drag, and put it back. Notifications reach a
	// subscriber in the order they were sent, so emitting first means the
	// window is released before any world update shows it moving.
	if err := h.plugin.EventsEmit(branchkit.EventsEmitRequest{EventType: snappedEventType, Data: snappedEvent(winID, direction, *frame)}); err != nil {
		branchkit.Logf("placement", "snap: emit %s: %v", snappedEventType, err)
	}

	frames := []branchkit.WindowFrame{
		{WindowID: winID, X: frame.X, Y: frame.Y, W: frame.W, H: frame.H},
	}
	readback := false
	if _, err := h.plugin.NativeBatchSetFrames(branchkit.NativeBatchSetFramesRequest{Frames: frames, Readback: &readback}); err != nil {
		return fmt.Errorf("snap: move the window: %w", err)
	}
	h.history.push(winID, branchkit.Rect{X: win.X, Y: win.Y, W: win.W, H: win.H})
	branchkit.Logf("placement", "snap: batch-set-frames succeeded (applied in %dms)", time.Since(start).Milliseconds())
	return nil
}

// usableArea is the part of a display a window may fill: the display minus
// the menu bar, Dock, taskbar or panels, as the OS reports it (NSScreen
// visibleFrame on macOS, the work area on Windows, _NET_WORKAREA on X11).
// A display that reports no visible bounds falls back to its full frame.
func usableArea(d branchkit.DisplayInfo) branchkit.Rect {
	if d.VisibleW > 0 && d.VisibleH > 0 {
		return branchkit.Rect{X: d.VisibleX, Y: d.VisibleY, W: d.VisibleW, H: d.VisibleH}
	}
	return branchkit.Rect{X: d.X, Y: d.Y, W: d.W, H: d.H}
}

func calculateSnapGeometry(win *branchkit.WindowInfo, screen branchkit.DisplayInfo, screenIdx int, displays []branchkit.DisplayInfo, direction string) *branchkit.Rect {
	a := usableArea(screen)
	switch direction {
	case "left":
		return &branchkit.Rect{X: a.X, Y: a.Y, W: a.W / 2, H: a.H}
	case "right":
		return &branchkit.Rect{X: a.X + a.W/2, Y: a.Y, W: a.W - a.W/2, H: a.H}
	case "top", "up":
		return &branchkit.Rect{X: a.X, Y: a.Y, W: a.W, H: a.H / 2}
	case "bottom", "down":
		return &branchkit.Rect{X: a.X, Y: a.Y + a.H/2, W: a.W, H: a.H - a.H/2}
	case "maximize", "full":
		return &branchkit.Rect{X: a.X, Y: a.Y, W: a.W, H: a.H}
	case "center":
		return &branchkit.Rect{X: a.X + a.W/4, Y: a.Y + a.H/4, W: a.W / 2, H: a.H / 2}
	case "next", "next monitor", "other screen", "move next",
		"prev", "previous monitor", "move back":
		if len(displays) < 2 {
			return nil
		}
		var nextIdx int
		if direction == "prev" || direction == "previous monitor" || direction == "move back" {
			nextIdx = (screenIdx + len(displays) - 1) % len(displays)
		} else {
			nextIdx = (screenIdx + 1) % len(displays)
		}
		t := usableArea(displays[nextIdx])

		// Same place relative to the usable area, so a window that filled
		// one display's usable area fills the next one's.
		relX := float64(win.X-a.X) / float64(a.W)
		relY := float64(win.Y-a.Y) / float64(a.H)
		relW := float64(win.W) / float64(a.W)
		relH := float64(win.H) / float64(a.H)

		return &branchkit.Rect{
			X: t.X + int(math.Round(relX*float64(t.W))),
			Y: t.Y + int(math.Round(relY*float64(t.H))),
			W: int(math.Round(relW * float64(t.W))),
			H: int(math.Round(relH * float64(t.H))),
		}
	default:
		return nil
	}
}
