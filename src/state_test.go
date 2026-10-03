package main

import (
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// "bring back window" returns the most recently minimized window that is
// still minimized; one the person already restored is skipped.
func TestBringBackSkipsWindowsAlreadyRestored(t *testing.T) {
	var m minimizedStack
	m.push("1")
	m.push("2")
	m.push("3")
	yes, no := true, false
	ws := []branchkit.WindowInfo{
		{ID: "1", IsMinimized: &yes},
		{ID: "2", IsMinimized: &yes},
		{ID: "3", IsMinimized: &no}, // restored from the Dock meanwhile
	}
	if id, ok := m.popStillMinimized(ws); !ok || id != "2" {
		t.Fatalf("got (%q, %v), want 2", id, ok)
	}
	if id, ok := m.popStillMinimized(ws); !ok || id != "1" {
		t.Fatalf("got (%q, %v), want 1", id, ok)
	}
	if _, ok := m.popStillMinimized(ws); ok {
		t.Fatal("stack should be empty")
	}
}
