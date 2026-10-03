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
	return h.place(activeWindowID, "snap", direction,
		func(win *branchkit.WindowInfo, screenIdx int, displays []branchkit.DisplayInfo) (*branchkit.Rect, error) {
			frame := calculateSnapGeometry(win, displays[screenIdx], screenIdx, displays, direction)
			if frame == nil {
				if (direction == "next" || direction == "prev") && len(displays) < 2 {
					return nil, fmt.Errorf("there is only one display")
				}
				return nil, fmt.Errorf("%q is not a position", direction)
			}
			return frame, nil
		})
}

// place moves one window to the frame `compute` picks for it: it finds
// the window (the given id, else the focused one) and the display its
// centre is on, announces the move, applies it, and records the old frame
// for "put it back". verb names the command in errors; position goes in
// the placement.snapped event.
func (h *Host) place(activeWindowID *string, verb, position string,
	compute func(win *branchkit.WindowInfo, screenIdx int, displays []branchkit.DisplayInfo) (*branchkit.Rect, error)) error {
	start := time.Now()
	wm, err := h.plugin.NativeWorldModel(branchkit.NativeWorldModelRequest{})
	if err != nil {
		return fmt.Errorf("%s: read the windows: %w", verb, err)
	}

	winID := ""
	if activeWindowID != nil {
		winID = *activeWindowID
	} else if wm.ActiveWindowID != nil {
		winID = *wm.ActiveWindowID
	}
	if winID == "" {
		return fmt.Errorf("%s: no window is focused", verb)
	}

	var win *branchkit.WindowInfo
	for i := range wm.Windows {
		if wm.Windows[i].ID == winID {
			win = &wm.Windows[i]
			break
		}
	}
	if win == nil {
		return fmt.Errorf("%s: window %s is not on screen", verb, winID)
	}
	if len(wm.Displays) == 0 {
		return fmt.Errorf("%s: no display is connected", verb)
	}
	screenIdx := displayOf(win, wm.Displays)

	frame, err := compute(win, screenIdx, wm.Displays)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, position, err)
	}
	branchkit.Logf("placement", "%s: window=%s position=%s → x=%d y=%d w=%d h=%d (from screen %d)",
		verb, winID, position, frame.X, frame.Y, frame.W, frame.H, screenIdx)

	// Say so BEFORE moving it. A plugin managing this window (a tiler)
	// takes it as "the user placed this one by hand" and lets go of it;
	// told after the move, it may already have seen the window leave its
	// slot, read that as a drag, and put it back. Notifications reach a
	// subscriber in the order they were sent, so emitting first means the
	// window is released before any world update shows it moving.
	if err := h.plugin.EventsEmit(branchkit.EventsEmitRequest{EventType: snappedEventType, Data: snappedEvent(winID, position, *frame)}); err != nil {
		branchkit.Logf("placement", "%s: emit %s: %v", verb, snappedEventType, err)
	}

	frames := []branchkit.WindowFrame{
		{WindowID: winID, X: frame.X, Y: frame.Y, W: frame.W, H: frame.H},
	}
	readback := false
	if _, err := h.plugin.NativeBatchSetFrames(branchkit.NativeBatchSetFramesRequest{Frames: frames, Readback: &readback}); err != nil {
		return fmt.Errorf("%s: move the window: %w", verb, err)
	}
	h.history.push(winID, branchkit.Rect{X: win.X, Y: win.Y, W: win.W, H: win.H})
	branchkit.Logf("placement", "%s: applied in %dms", verb, time.Since(start).Milliseconds())
	return nil
}

// displayOf is the index of the display holding the window's centre, or 0.
func displayOf(win *branchkit.WindowInfo, displays []branchkit.DisplayInfo) int {
	cx, cy := win.X+win.W/2, win.Y+win.H/2
	for i, d := range displays {
		if cx >= d.X && cx < d.X+d.W && cy >= d.Y && cy < d.Y+d.H {
			return i
		}
	}
	return 0
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
	case "top":
		return &branchkit.Rect{X: a.X, Y: a.Y, W: a.W, H: a.H / 2}
	case "bottom":
		return &branchkit.Rect{X: a.X, Y: a.Y + a.H/2, W: a.W, H: a.H - a.H/2}
	case "maximize":
		return &branchkit.Rect{X: a.X, Y: a.Y, W: a.W, H: a.H}
	case "almost_maximize":
		// 90% of the usable area, centred: the desktop shows round the edge.
		dx, dy := a.W/20, a.H/20
		return &branchkit.Rect{X: a.X + dx, Y: a.Y + dy, W: a.W - 2*dx, H: a.H - 2*dy}
	case "center":
		return &branchkit.Rect{X: a.X + a.W/4, Y: a.Y + a.H/4, W: a.W / 2, H: a.H / 2}
	case "left_third", "center_third", "right_third", "left_two_thirds", "right_two_thirds":
		// The last third takes the leftover pixels, so a third and the
		// two-thirds beside it tile the area exactly.
		t := a.W / 3
		switch direction {
		case "left_third":
			return &branchkit.Rect{X: a.X, Y: a.Y, W: t, H: a.H}
		case "center_third":
			return &branchkit.Rect{X: a.X + t, Y: a.Y, W: t, H: a.H}
		case "right_third":
			return &branchkit.Rect{X: a.X + 2*t, Y: a.Y, W: a.W - 2*t, H: a.H}
		case "left_two_thirds":
			return &branchkit.Rect{X: a.X, Y: a.Y, W: 2 * t, H: a.H}
		default: // right_two_thirds
			return &branchkit.Rect{X: a.X + t, Y: a.Y, W: a.W - t, H: a.H}
		}
	case "top_left", "top_right", "bottom_left", "bottom_right":
		hw, hh := a.W/2, a.H/2
		r := branchkit.Rect{X: a.X, Y: a.Y, W: hw, H: hh}
		if direction == "top_right" || direction == "bottom_right" {
			r.X, r.W = a.X+hw, a.W-hw
		}
		if direction == "bottom_left" || direction == "bottom_right" {
			r.Y, r.H = a.Y+hh, a.H-hh
		}
		return &r
	case "next", "prev":
		if len(displays) < 2 {
			return nil
		}
		var nextIdx int
		if direction == "prev" {
			nextIdx = (screenIdx + len(displays) - 1) % len(displays)
		} else {
			nextIdx = (screenIdx + 1) % len(displays)
		}
		return mapBetween(win, a, usableArea(displays[nextIdx]))
	default:
		return nil
	}
}

// mapBetween puts the window at the same place relative to the target
// usable area as it has in the source one, so a window that filled one
// display's usable area fills the other's.
func mapBetween(win *branchkit.WindowInfo, a, t branchkit.Rect) *branchkit.Rect {
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
}
