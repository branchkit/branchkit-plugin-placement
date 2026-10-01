package main

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/branchkit/plugin-sdk-go"
)

// --- Per-action handlers ---
//
// Param structs (SnapParams, MoveToSpaceParams, …) AND their registrars
// (HandleSnap, HandleMoveToSpace, …) live in actions_gen.go, generated from
// plugin.json's action_types block — so a handler's params are typed without
// unmarshaling by hand, and the action string is never spelled here at all.
// Register in main.go with the generated Handle<Action>, not the untyped
// plugin.HandleAction. Slot captures from voice
// commands (`<number>`, `<text>`) substitute into action params via the
// matching engine's template syntax (`"space": "{number}"`) which always produces
// strings — so integer-valued params stay typed as strings in the manifest
// and are parsed inside the handler with strconv.

func (h *Host) handleDeskSwitch(p DeskSwitchParams, _ *branchkit.OnActionRequest) (any, error) {
	space, err := strconv.Atoi(p.Space)
	// macOS numbers its "Switch to Desktop N" hotkeys 1-16; elsewhere the
	// platform checks the number against the desktops that exist.
	if err != nil || space < 1 || (runtime.GOOS == "darwin" && space > 16) {
		return nil, fmt.Errorf("desk_switch: %q is not a desktop number", p.Space)
	}
	// The actuator resolves the user's actual "Switch to Desktop N" symbolic
	// hotkey (respects remaps, auto-enables disabled shortcuts) — no
	// hardcoded Ctrl+N keycode map.
	return nil, h.switchToDesktop(space)
}

func (h *Host) handleWindowsSnap(p SnapParams, req *branchkit.OnActionRequest) (any, error) {
	if p.Position == nil {
		return nil, fmt.Errorf("snap: no position given")
	}
	return nil, h.handleSnap(req.ActiveWindowID, string(*p.Position))
}

func (h *Host) handleWindowsMoveToSpace(p MoveToSpaceParams, req *branchkit.OnActionRequest) (any, error) {
	space, err := strconv.Atoi(p.Space)
	if err != nil || space < 1 {
		return nil, fmt.Errorf("move_to_space: %q is not a desktop number", p.Space)
	}
	// Explicit window_id wins over the envelope's active window — a
	// dispatching plugin (browser tab-to-desk) targets a window it just
	// created, which may not have claimed focus by the time this lands.
	windowID := req.ActiveWindowID
	if p.WindowID != nil && *p.WindowID != "" {
		windowID = p.WindowID
	}
	return nil, h.handleMoveToSpace(windowID, space, p.Stay != nil && *p.Stay)
}
