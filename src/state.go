package main

import (
	"fmt"
	"sync"

	"github.com/branchkit/plugin-sdk-go"
)

// minimizedStack remembers the windows this plugin minimized, most recent
// last, so "bring back window" knows which one is meant.
type minimizedStack struct {
	mu  sync.Mutex
	ids []string
}

func (m *minimizedStack) push(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ids = append(m.ids, id)
	if len(m.ids) > maxUndo {
		m.ids = m.ids[len(m.ids)-maxUndo:]
	}
}

// popStillMinimized returns the most recent window that is still
// minimized (one the person restored another way is skipped).
func (m *minimizedStack) popStillMinimized(windows []branchkit.WindowInfo) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	minimized := map[string]bool{}
	for _, w := range windows {
		if w.IsMinimized != nil && *w.IsMinimized {
			minimized[w.ID] = true
		}
	}
	for len(m.ids) > 0 {
		id := m.ids[len(m.ids)-1]
		m.ids = m.ids[:len(m.ids)-1]
		if minimized[id] {
			return id, true
		}
	}
	return "", false
}

// targetWindow is the window a state command acts on: the named app's
// frontmost window, else the focused one.
func (h *Host) targetWindow(verb string, app *string, req *branchkit.OnActionRequest) (string, error) {
	if app != nil && *app != "" {
		id, err := h.namedWindow(*app)
		if err != nil {
			return "", fmt.Errorf("%s: %w", verb, err)
		}
		return id, nil
	}
	if req.ActiveWindowID != nil && *req.ActiveWindowID != "" {
		return *req.ActiveWindowID, nil
	}
	if f, err := h.plugin.NativeFocusedWindowID(); err == nil && f != nil && f.WindowID != "" {
		return f.WindowID, nil
	}
	return "", fmt.Errorf("%s: no window is focused", verb)
}

func (h *Host) handleMinimize(p MinimizeParams, req *branchkit.OnActionRequest) (any, error) {
	id, err := h.targetWindow("minimize", p.App, req)
	if err != nil {
		return nil, err
	}
	if err := h.plugin.NativeMinimizeWindow(branchkit.NativeMinimizeWindowRequest{WindowID: id}); err != nil {
		return nil, fmt.Errorf("minimize: %w", err)
	}
	h.minimized.push(id)
	return nil, nil
}

func (h *Host) handleBringBack(p BringBackParams, req *branchkit.OnActionRequest) (any, error) {
	wm, err := h.plugin.NativeWorldModel(branchkit.NativeWorldModelRequest{})
	if err != nil {
		return nil, fmt.Errorf("bring back: read the windows: %w", err)
	}
	id := ""
	if p.App != nil && *p.App != "" {
		id = firstMinimized(wm.Windows, func(w branchkit.WindowInfo) bool { return w.AppID == *p.App })
		if id == "" {
			return nil, fmt.Errorf("bring back: %s has no minimized window", *p.App)
		}
	} else if last, ok := h.minimized.popStillMinimized(wm.Windows); ok {
		id = last
	} else {
		// Nothing minimized by voice: the focused app's minimized window.
		app := ""
		if wm.ActiveWindowID != nil {
			for _, w := range wm.Windows {
				if w.ID == *wm.ActiveWindowID {
					app = w.AppID
				}
			}
		}
		id = firstMinimized(wm.Windows, func(w branchkit.WindowInfo) bool { return app != "" && w.AppID == app })
		if id == "" {
			return nil, fmt.Errorf("bring back: there is no minimized window to bring back")
		}
	}
	if err := h.plugin.NativeUnminimizeWindow(branchkit.NativeUnminimizeWindowRequest{WindowID: id}); err != nil {
		return nil, fmt.Errorf("bring back: %w", err)
	}
	if err := h.plugin.NativeRaiseWindow(branchkit.NativeRaiseWindowRequest{WindowID: id}); err != nil {
		branchkit.Logf("placement", "bring back: raise %s: %v", id, err)
	}
	return nil, nil
}

func firstMinimized(windows []branchkit.WindowInfo, match func(branchkit.WindowInfo) bool) string {
	for _, w := range windows {
		if w.IsMinimized != nil && *w.IsMinimized && match(w) {
			return w.ID
		}
	}
	return ""
}

func (h *Host) handleFullscreen(p FullscreenParams, req *branchkit.OnActionRequest) (any, error) {
	id, err := h.targetWindow("fullscreen", p.App, req)
	if err != nil {
		return nil, err
	}
	if err := h.plugin.NativeToggleFullscreen(branchkit.NativeToggleFullscreenRequest{WindowID: id}); err != nil {
		return nil, fmt.Errorf("fullscreen: %w", err)
	}
	return nil, nil
}

func (h *Host) handlePin(p PinParams, req *branchkit.OnActionRequest) (any, error) {
	id, err := h.targetWindow("pin", p.App, req)
	if err != nil {
		return nil, err
	}
	pinned := p.Pinned == nil || *p.Pinned
	if err := h.plugin.NativePinWindowAbove(branchkit.NativePinWindowAboveRequest{WindowID: id, Pinned: pinned}); err != nil {
		return nil, fmt.Errorf("pin: %w", err)
	}
	return nil, nil
}

func (h *Host) handleClose(p CloseParams, req *branchkit.OnActionRequest) (any, error) {
	id, err := h.targetWindow("close", p.App, req)
	if err != nil {
		return nil, err
	}
	closed, err := h.plugin.NativeCloseWindow(branchkit.NativeCloseWindowRequest{WindowID: id})
	if err != nil {
		return nil, fmt.Errorf("close: %w", err)
	}
	if !closed {
		return nil, fmt.Errorf("close: the window did not close (it may be asking to save)")
	}
	return nil, nil
}
