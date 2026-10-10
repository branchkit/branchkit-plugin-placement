package main

import "github.com/branchkit/plugin-sdk-go"

// Host is what every handler in this plugin needs: the platform handle.
// Handlers are methods on it, so their dependency is visible in the
// signature and cannot be read before it exists.
type Host struct {
	plugin platformCalls
	// history holds each window's frames from before its placements, for
	// "put it back".
	history *frameHistory
	// minimized is the windows minimized by voice, for "bring back window".
	minimized *minimizedStack
}

// platformCalls is the part of *branchkit.Plugin the handlers call. It is
// an interface so a test can stand in for the platform and see which
// window a handler acted on; in the running plugin it is always the
// *branchkit.Plugin main creates.
type platformCalls interface {
	EventsEmit(branchkit.EventsEmitRequest) error
	InputMouseButton(branchkit.InputMouseButtonRequest) error
	InputPressKey(branchkit.InputPressKeyRequest) error
	NativeBatchSetFrames(branchkit.NativeBatchSetFramesRequest) ([]branchkit.WindowFrame, error)
	NativeCloseWindow(branchkit.NativeCloseWindowRequest) (bool, error)
	NativeCursorInfo() (*branchkit.NativeCursorInfoResponse, error)
	NativeFocusedWindowID() (*branchkit.NativeFocusedWindowIDResponse, error)
	NativeListSpaces() ([]branchkit.SpaceInfo, error)
	NativeMinimizeWindow(branchkit.NativeMinimizeWindowRequest) error
	NativeMoveWindowToSpace(branchkit.NativeMoveWindowToSpaceRequest) (bool, error)
	NativePinWindowAbove(branchkit.NativePinWindowAboveRequest) error
	NativeRaiseWindow(branchkit.NativeRaiseWindowRequest) error
	NativeSwitchSpace(branchkit.NativeSwitchSpaceRequest) error
	NativeToggleFullscreen(branchkit.NativeToggleFullscreenRequest) error
	NativeUnminimizeWindow(branchkit.NativeUnminimizeWindowRequest) error
	NativeWarpCursor(branchkit.NativeWarpCursorRequest) error
	NativeWindowBounds(branchkit.NativeWindowBoundsRequest) (*branchkit.NativeWindowBoundsResponse, error)
	NativeWorldModel(branchkit.NativeWorldModelRequest) (*branchkit.WorldModel, error)
}

func newHost(p *branchkit.Plugin) *Host {
	return &Host{plugin: p, history: newFrameHistory(), minimized: &minimizedStack{}}
}
