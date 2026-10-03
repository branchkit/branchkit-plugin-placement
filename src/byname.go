package main

import (
	"fmt"

	"github.com/branchkit/plugin-sdk-go"
)

// appWindow picks the window to act on for a named app: its frontmost
// window that is not minimized, in the world model's front-to-back order.
// app is the id the apps collection substitutes (bundle id on macOS, the
// app id elsewhere), which is what WindowInfo.AppID carries.
func appWindow(windows []branchkit.WindowInfo, app string) (string, error) {
	minimized := ""
	for _, w := range windows {
		if w.AppID != app {
			continue
		}
		if w.IsMinimized != nil && *w.IsMinimized {
			if minimized == "" {
				minimized = w.ID
			}
			continue
		}
		return w.ID, nil
	}
	if minimized != "" {
		return "", fmt.Errorf("%s's windows are all minimized", app)
	}
	return "", fmt.Errorf("%s has no window open", app)
}

// namedWindow resolves an app named in a command to the window to act on.
func (h *Host) namedWindow(app string) (string, error) {
	wm, err := h.plugin.NativeWorldModel(branchkit.NativeWorldModelRequest{})
	if err != nil {
		return "", fmt.Errorf("read the windows: %w", err)
	}
	return appWindow(wm.Windows, app)
}
