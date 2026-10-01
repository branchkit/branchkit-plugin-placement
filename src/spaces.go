package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/branchkit/plugin-sdk-go"
)

// Timing constants for Mission Control space transitions.
// These values account for macOS animation latency between operations.
const (
	cursorSettleDelay  = 50 * time.Millisecond  // wait for cursor warp to register
	mouseDownHoldDelay = 25 * time.Millisecond  // hold before space switch keystroke
	spaceTransitDelay  = 200 * time.Millisecond // wait for Mission Control space animation
)

// switchToDesktop switches to a Mission Control desktop by number via the
// actuator, which resolves the user's own "Switch to Desktop N" symbolic
// hotkey (respecting remaps and auto-enabling disabled shortcuts) instead of
// assuming Ctrl+N. Desktops 1-16.
func (h *Host) switchToDesktop(desktop int) {
	branchkit.Logf("windows", "switch_space → desktop %d", desktop)
	if err := h.plugin.NativeSwitchSpace(branchkit.NativeSwitchSpaceRequest{SpaceID: desktop}); err != nil {
		branchkit.Logf("windows", "switch to desktop %d: %v", desktop, err)
	}
}

// cursorPosition returns the current cursor location, or ok=false.
func (h *Host) cursorPosition() (x, y int, ok bool) {
	info, err := h.plugin.NativeCursorInfo()
	if err != nil {
		return 0, 0, false
	}
	return info.X, info.Y, true
}

// originDesktopOrdinal returns the Mission Control desktop number (the Ctrl+N
// index, counted across displays in managed-display order, matching the
// desk_switch convention) the user is looking at on the display containing
// the given point — the desk to hop back to after a stay move.
//
// Derived from the display's ACTIVE space, deliberately NOT from
// window↔space membership: the window being moved is by construction on the
// user's current space (the drag grab requires it under the cursor), and a
// freshly created window (the browser plugin's tab-to-desk pop path) can lag
// CGS membership queries — the old membership-based derivation hopped the
// user to the wrong desk (2026-07-25). Falls back to the first active user
// space when no display contains the point. Returns 0 only when spaces can't
// be listed.
func (h *Host) originDesktopOrdinal(displays []branchkit.DisplayInfo, pointX, pointY int) int {
	spaces, err := h.plugin.NativeListSpaces()
	if err != nil {
		branchkit.Logf("windows", "move-to-space: list spaces: %v", err)
		return 0
	}
	displayID := 0
	for _, d := range displays {
		if pointX >= d.X && pointX < d.X+d.W && pointY >= d.Y && pointY < d.Y+d.H {
			displayID = d.ID
			break
		}
	}
	ordinal, firstActive, matched := 0, 0, 0
	var order []string
	for _, s := range spaces {
		if s.SpaceType != "user" {
			continue
		}
		ordinal++
		order = append(order, fmt.Sprintf("%d:d%d:%v", s.SpaceID, s.DisplayID, s.IsActive))
		if !s.IsActive {
			continue
		}
		if firstActive == 0 {
			firstActive = ordinal
		}
		if matched == 0 && s.DisplayID == displayID {
			matched = ordinal
		}
	}
	result := matched
	if result == 0 {
		result = firstActive
	}
	branchkit.Logf("windows", "move-to-space: origin desk=%d (window display=%d, spaces=%s)",
		result, displayID, strings.Join(order, " "))
	return result
}

// handleMoveToSpace moves the active window to the given Mission Control space
// by holding the title bar with the mouse, pressing Ctrl+N, then releasing —
// which inherently navigates to the target space with the window. With `stay`,
// it hops back to the origin desktop after delivery (the private CGS
// move-without-switching APIs are dead on modern macOS — verified silent no-op
// on Sequoia 2026-07-25 — so a visible round trip is the only non-SIP path).
// movedToSpaceEventType is emitted when a window is sent to another
// desktop, before it moves.
const movedToSpaceEventType = "windows.moved_to_space"

func movedToSpaceEvent(windowID string, space int, stay bool) json.RawMessage {
	b, _ := json.Marshal(map[string]any{"window_id": windowID, "space": space, "stay": stay})
	return b
}

func (h *Host) handleMoveToSpace(activeWindowID *string, space int, stay bool) {
	if space < 1 || space > 16 {
		branchkit.Logf("windows", "move-to-space: invalid space %d", space)
		return
	}

	wm, err := h.plugin.NativeWorldModel(branchkit.NativeWorldModelRequest{})
	if err != nil {
		branchkit.Logf("windows", "move-to-space: get world model: %v", err)
		return
	}

	winID := ""
	if activeWindowID != nil {
		winID = *activeWindowID
	} else if wm.ActiveWindowID != nil {
		winID = *wm.ActiveWindowID
	}

	var winX, winY, winW, winH int
	found := false
	for _, w := range wm.Windows {
		if w.ID == winID {
			winX = w.X
			winY = w.Y
			winW = w.W
			winH = w.H
			found = true
			break
		}
	}

	// Fallback: ask for the window's bounds directly (the world model can
	// miss a window that just appeared). Without an id, the focused one.
	if !found {
		if winID == "" {
			if focused, err := h.plugin.NativeFocusedWindowID(); err == nil && focused != nil {
				winID = focused.WindowID
			}
		}
		if winID != "" {
			if b, err := h.plugin.NativeWindowBounds(branchkit.NativeWindowBoundsRequest{WindowID: winID}); err == nil {
				winX, winY, winW, winH = b.X, b.Y, b.W, b.H
				found = true
			}
		}
	}

	if !found {
		branchkit.Logf("windows", "move-to-space: could not find window position")
		return
	}

	// Before the move, for the same reason as windows.snapped: a plugin
	// placing this window must let go of it before it sees it leave.
	if err := h.plugin.EventsEmit(branchkit.EventsEmitRequest{EventType: movedToSpaceEventType, Data: movedToSpaceEvent(winID, space, stay)}); err != nil {
		branchkit.Logf("windows", "move-to-space: emit %s: %v", movedToSpaceEventType, err)
	}

	// Resolve the return desktop BEFORE the move — afterwards the window (and
	// we, riding along with it) are on the target space.
	returnOrdinal := 0
	if stay {
		returnOrdinal = h.originDesktopOrdinal(wm.Displays, winX+winW/2, winY+winH/2)
		if returnOrdinal == 0 {
			branchkit.Logf("windows", "move-to-space: stay requested but origin desktop unknown — will follow instead")
		} else if returnOrdinal == space {
			returnOrdinal = 0 // already there; nothing to hop back to
		}
	}

	// Remember where the cursor was so the whole operation doesn't strand it
	// on the moved window's title bar.
	origCursorX, origCursorY, restoreCursor := h.cursorPosition()

	// Click title bar area, hold, switch space, release
	clickX := winX + 75
	clickY := winY + 10

	// Warp cursor to title bar
	if err := h.plugin.NativeWarpCursor(branchkit.NativeWarpCursorRequest{X: clickX, Y: clickY}); err != nil {
		branchkit.Logf("windows", "move-to-space: warp cursor: %v", err)
		return
	}
	time.Sleep(cursorSettleDelay)

	// Mouse down, then a zero-distance drag to latch the window onto the
	// cursor — macOS only treats the window as grabbed once a dragged event
	// follows the press (the Amethyst/Silica latch).
	//
	// The release is deferred, not just written below, because the button we
	// press here is REAL system state that outlives this function. A panic
	// between press and release is caught by the SDK's per-request recovery
	// (plugin-sdk-go/rpc.go handleRequest), so the process keeps running and
	// nothing ever posts the matching up event — the window server stays
	// latched with the left button down and every subsequent click reads as a
	// drag, until the user physically clicks. The plugin never notices.
	//
	// Same rule as the boot-time tag releases: the side that outlives the
	// failure must own the release — and here that side is the OS, so the
	// release has to be unconditional.
	//
	// A backstop rather than a plain `defer mouseButton("release")`, because
	// ORDER is load-bearing on the happy path: the drop has to land before the
	// return-hop below, so the release can't be moved to function exit. The
	// latch releases exactly once, wherever it happens first.
	releaseMouse := releaseOnce(func() { h.mouseButton("release") })
	h.mouseButton("press")
	defer releaseMouse()
	h.mouseButton("drag")
	time.Sleep(mouseDownHoldDelay)

	// Switch to the target desktop from under the held window (symbolic
	// hotkey — respects the user's actual shortcut config)
	h.switchToDesktop(space)

	time.Sleep(spaceTransitDelay)

	// Mouse up — here, not at function exit, so the drop lands before any
	// return-hop.
	releaseMouse()

	// Stay variant: hop back to the origin desktop once the drop has landed.
	if returnOrdinal != 0 {
		time.Sleep(spaceTransitDelay)
		h.switchToDesktop(returnOrdinal)
	}

	if restoreCursor {
		if err := h.plugin.NativeWarpCursor(branchkit.NativeWarpCursorRequest{X: origCursorX, Y: origCursorY}); err != nil {
			branchkit.Logf("windows", "cursor restore: %v", err)
		}
	}
}

// releaseOnce wraps a release so it runs at most once, however many paths reach
// it. Lets a sequence keep its release in the position the happy path needs
// while still having `defer` as the backstop for the paths that never get
// there — a plain `defer release()` would move the release to function exit,
// which is wrong when later steps depend on it having already landed.
func releaseOnce(fn func()) func() {
	done := false
	return func() {
		if done {
			return
		}
		done = true
		fn()
	}
}

// mouseButton presses, releases, or drag-latches the left mouse button via
// the input.mouse_button RPC. (The old raw `dispatch` route is denied to
// plugin callers by the operation auth layer — the grab half of the drag
// trick had been failing silently through it.)
func (h *Host) mouseButton(direction string) {
	left := "left"
	if err := h.plugin.InputMouseButton(branchkit.InputMouseButtonRequest{Direction: direction, Button: &left}); err != nil {
		branchkit.Logf("windows", "mouse_button %s: %v", direction, err)
	}
}
