package main

import (
	"github.com/branchkit/plugin-sdk-go"
)

func main() {
	p := branchkit.NewPlugin()
	h := newHost(p)

	// Registrars are generated from plugin.json into actions_gen.go, so the
	// action string and the params type both come from the manifest and
	// neither can drift from it.
	HandleSnap(p, h.handleWindowsSnap)
	HandleDeskSwitch(p, h.handleDeskSwitch)
	HandleMoveToSpace(p, h.handleWindowsMoveToSpace)
	HandleOverview(p, h.handleOverview)
	HandleCycleWindow(p, h.handleCycleWindow)
	HandleUndo(p, h.handleUndo)

	p.Run()
}
