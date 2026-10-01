package main

import (
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// Bad input is an error the caller sees, never a silent success. These
// return before any platform call, so a Host with no plugin is enough.
func TestHandlersRejectBadInputWithAnError(t *testing.T) {
	h := &Host{}
	req := &branchkit.OnActionRequest{}

	for _, space := range []string{"", "zero", "0", "-2"} {
		if _, err := h.handleDeskSwitch(DeskSwitchParams{Space: space}, req); err == nil {
			t.Errorf("desk_switch %q: want an error", space)
		}
		if _, err := h.handleWindowsMoveToSpace(MoveToSpaceParams{Space: space}, req); err == nil {
			t.Errorf("move_to_space %q: want an error", space)
		}
	}
	if _, err := h.handleWindowsSnap(SnapParams{}, req); err == nil {
		t.Error("snap with no position: want an error")
	}
}
