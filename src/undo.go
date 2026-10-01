package main

import (
	"fmt"
	"sync"

	"github.com/branchkit/plugin-sdk-go"
)

// maxUndo is how many placements of one window "put it back" can walk
// back through.
const maxUndo = 10

// frameHistory is each window's frames from before its placements, most
// recent last. It lives only as long as the plugin process: "put it back"
// is for the moves you just made, not a record to keep.
type frameHistory struct {
	mu       sync.Mutex
	byWindow map[string][]branchkit.Rect
}

func newFrameHistory() *frameHistory {
	return &frameHistory{byWindow: map[string][]branchkit.Rect{}}
}

func (f *frameHistory) push(windowID string, r branchkit.Rect) {
	f.mu.Lock()
	defer f.mu.Unlock()
	h := append(f.byWindow[windowID], r)
	if len(h) > maxUndo {
		h = h[len(h)-maxUndo:]
	}
	f.byWindow[windowID] = h
}

func (f *frameHistory) pop(windowID string) (branchkit.Rect, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	h := f.byWindow[windowID]
	if len(h) == 0 {
		return branchkit.Rect{}, false
	}
	r := h[len(h)-1]
	f.byWindow[windowID] = h[:len(h)-1]
	return r, true
}

// handleUndo puts the focused window back where it was before its last
// placement.
func (h *Host) handleUndo(req *branchkit.OnActionRequest) (any, error) {
	winID := ""
	if req.ActiveWindowID != nil {
		winID = *req.ActiveWindowID
	} else if focused, err := h.plugin.NativeFocusedWindowID(); err == nil && focused != nil {
		winID = focused.WindowID
	}
	if winID == "" {
		return nil, fmt.Errorf("put it back: no window is focused")
	}
	prev, ok := h.history.pop(winID)
	if !ok {
		return nil, fmt.Errorf("put it back: this window has not been placed")
	}
	// Announced like any placement, so a tiler lets go of the window first.
	if err := h.plugin.EventsEmit(branchkit.EventsEmitRequest{EventType: snappedEventType, Data: snappedEvent(winID, "restore", prev)}); err != nil {
		branchkit.Logf("placement", "undo: emit %s: %v", snappedEventType, err)
	}
	readback := false
	frames := []branchkit.WindowFrame{{WindowID: winID, X: prev.X, Y: prev.Y, W: prev.W, H: prev.H}}
	if _, err := h.plugin.NativeBatchSetFrames(branchkit.NativeBatchSetFramesRequest{Frames: frames, Readback: &readback}); err != nil {
		h.history.push(winID, prev) // still there to try again
		return nil, fmt.Errorf("put it back: move the window: %w", err)
	}
	return nil, nil
}
