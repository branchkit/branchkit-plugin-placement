package main

import (
	"strings"
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

func TestAppWindowPicksTheFrontmostUnminimizedWindow(t *testing.T) {
	yes, no := true, false
	ws := []branchkit.WindowInfo{
		{ID: "1", AppID: "com.other"},
		{ID: "2", AppID: "com.slack", IsMinimized: &yes},
		{ID: "3", AppID: "com.slack", IsMinimized: &no},
		{ID: "4", AppID: "com.slack"},
	}
	if id, err := appWindow(ws, "com.slack"); err != nil || id != "3" {
		t.Fatalf("got (%q, %v), want 3", id, err)
	}
}

func TestAppWindowSaysWhyThereIsNone(t *testing.T) {
	yes := true
	ws := []branchkit.WindowInfo{{ID: "2", AppID: "com.slack", IsMinimized: &yes}}
	if _, err := appWindow(ws, "com.slack"); err == nil || !strings.Contains(err.Error(), "minimized") {
		t.Errorf("all minimized: err = %v", err)
	}
	if _, err := appWindow(ws, "com.mail"); err == nil || !strings.Contains(err.Error(), "no window open") {
		t.Errorf("not open: err = %v", err)
	}
}
