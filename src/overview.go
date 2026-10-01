package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"

	"github.com/branchkit/plugin-sdk-go"
)

// overviewKey is the key each OS shows every window with: Mission Control
// (Ctrl+Up) on macOS, Task View (Win+Tab) on Windows, and on Linux the
// Super key, which opens GNOME's Activities and KDE's overview.
func overviewKey(goos string) (name string, modifiers []string) {
	switch goos {
	case "darwin":
		return "up", []string{"ctrl"}
	case "windows":
		return "tab", []string{"meta"}
	default:
		return "meta", nil
	}
}

func (h *Host) handleOverview(_ *branchkit.OnActionRequest) (any, error) {
	name, mods := overviewKey(runtime.GOOS)
	if err := h.plugin.InputPressKey(branchkit.InputPressKeyRequest{Name: &name, Modifiers: mods}); err != nil {
		return nil, fmt.Errorf("show all windows: %w", err)
	}
	return nil, nil
}

func (h *Host) handleCycleWindow(req *branchkit.OnActionRequest) (any, error) {
	wm, err := h.plugin.NativeWorldModel(branchkit.NativeWorldModelRequest{})
	if err != nil {
		return nil, fmt.Errorf("next window: read the windows: %w", err)
	}
	active := ""
	if req.ActiveWindowID != nil {
		active = *req.ActiveWindowID
	} else if wm.ActiveWindowID != nil {
		active = *wm.ActiveWindowID
	}
	next, err := nextAppWindow(wm.Windows, active)
	if err != nil {
		return nil, fmt.Errorf("next window: %w", err)
	}
	if err := h.plugin.NativeRaiseWindow(branchkit.NativeRaiseWindowRequest{WindowID: next}); err != nil {
		return nil, fmt.Errorf("next window: raise it: %w", err)
	}
	return nil, nil
}

// nextAppWindow picks the window after `active` among the same app's
// windows, in a fixed order (by id), so saying it again walks through all
// of them and comes back round. Minimized windows are skipped.
func nextAppWindow(windows []branchkit.WindowInfo, active string) (string, error) {
	var cur *branchkit.WindowInfo
	for i := range windows {
		if windows[i].ID == active {
			cur = &windows[i]
			break
		}
	}
	if cur == nil {
		return "", fmt.Errorf("no window is focused")
	}
	var ids []string
	for _, w := range windows {
		if !sameApp(w, *cur) || (w.IsMinimized != nil && *w.IsMinimized) {
			continue
		}
		ids = append(ids, w.ID)
	}
	if len(ids) < 2 {
		return "", fmt.Errorf("%s has no other window", appLabel(*cur))
	}
	sort.Slice(ids, func(i, j int) bool { return idLess(ids[i], ids[j]) })
	for i, id := range ids {
		if id == active {
			return ids[(i+1)%len(ids)], nil
		}
	}
	return ids[0], nil
}

func sameApp(a, b branchkit.WindowInfo) bool {
	if a.AppID != "" || b.AppID != "" {
		return a.AppID == b.AppID
	}
	return a.AppName == b.AppName
}

func appLabel(w branchkit.WindowInfo) string {
	if w.AppName != "" {
		return w.AppName
	}
	return "this app"
}

// idLess orders numeric ids (macOS, X11, Windows handles) as numbers and
// anything else as text.
func idLess(a, b string) bool {
	x, errA := strconv.ParseInt(a, 10, 64)
	y, errB := strconv.ParseInt(b, 10, 64)
	if errA == nil && errB == nil {
		return x < y
	}
	return a < b
}
