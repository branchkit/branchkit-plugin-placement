package main

import (
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// Each window walks back through its own placements, most recent first,
// and no further than maxUndo.
func TestFrameHistoryIsPerWindowAndBounded(t *testing.T) {
	f := newFrameHistory()
	for i := 0; i < maxUndo+3; i++ {
		f.push("a", branchkit.Rect{X: i})
	}
	f.push("b", branchkit.Rect{X: 99})

	for want := maxUndo + 2; want >= 3; want-- {
		r, ok := f.pop("a")
		if !ok || r.X != want {
			t.Fatalf("pop a: got (%d, %v), want %d", r.X, ok, want)
		}
	}
	if _, ok := f.pop("a"); ok {
		t.Fatal("a: history should be exhausted after maxUndo entries")
	}
	if r, ok := f.pop("b"); !ok || r.X != 99 {
		t.Fatalf("b's history must be its own: got (%d, %v)", r.X, ok)
	}
}
