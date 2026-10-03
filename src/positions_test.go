package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// The manifest's position enum, the geometry and the voice commands must
// agree: every declared position has geometry, and every snap command
// names a declared position.
func TestPositionsAgreeAcrossManifestGeometryAndCommands(t *testing.T) {
	var m struct {
		ActionTypes map[string]struct {
			Fields []struct {
				Key        string   `json:"key"`
				EnumValues []string `json:"enum_values"`
			} `json:"fields"`
		} `json:"action_types"`
	}
	raw, err := os.ReadFile("../plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for _, f := range m.ActionTypes["snap"].Fields {
		if f.Key == "position" {
			for _, v := range f.EnumValues {
				declared[v] = true
			}
		}
	}
	win, d, _ := makeTestWorld()
	two := []branchkit.DisplayInfo{d, {ID: 2, X: 1920, Y: 0, W: 1920, H: 1080}}
	for pos := range declared {
		if calculateSnapGeometry(win, d, 0, two, pos) == nil {
			t.Errorf("declared position %q has no geometry", pos)
		}
	}

	var cmds []struct {
		Pattern []any `json:"pattern"`
		Action  struct {
			Type   string         `json:"type"`
			Params map[string]any `json:"params"`
		} `json:"action"`
	}
	raw, err = os.ReadFile("../commands.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &cmds); err != nil {
		t.Fatal(err)
	}
	for _, c := range cmds {
		if c.Action.Type != "placement.snap" {
			continue
		}
		pos, _ := c.Action.Params["position"].(string)
		if !declared[pos] {
			t.Errorf("command %v snaps to %q, which the manifest does not declare", c.Pattern, pos)
		}
	}
}

// Thirds and their two-thirds neighbours, and the four quarters, tile the
// usable area with no gap or overlap, odd sizes included.
func TestThirdsAndQuartersTileTheArea(t *testing.T) {
	win, _, _ := makeTestWorld()
	d := branchkit.DisplayInfo{ID: 1, X: 0, Y: 0, W: 1001, H: 701,
		VisibleX: 10, VisibleY: 25, VisibleW: 991, VisibleH: 675}
	g := func(p string) *branchkit.Rect { return calculateSnapGeometry(win, d, 0, []branchkit.DisplayInfo{d}, p) }

	l, c, r := g("left_third"), g("center_third"), g("right_third")
	if l.X != 10 || l.X+l.W != c.X || c.X+c.W != r.X || r.X+r.W != 10+991 {
		t.Errorf("thirds: %+v %+v %+v", *l, *c, *r)
	}
	if lt := g("left_two_thirds"); lt.X+lt.W != r.X {
		t.Errorf("left two thirds must meet the right third: %+v vs %+v", *lt, *r)
	}
	if rt := g("right_two_thirds"); rt.X != l.X+l.W || rt.X+rt.W != 10+991 {
		t.Errorf("right two thirds must meet the left third: %+v vs %+v", *rt, *l)
	}

	tl, tr, bl, br := g("top_left"), g("top_right"), g("bottom_left"), g("bottom_right")
	area := 0
	for _, q := range []*branchkit.Rect{tl, tr, bl, br} {
		area += q.W * q.H
	}
	if area != 991*675 || tl.X+tl.W != tr.X || tl.Y+tl.H != bl.Y || br.X+br.W != 10+991 || br.Y+br.H != 25+675 {
		t.Errorf("quarters do not tile: %+v %+v %+v %+v", *tl, *tr, *bl, *br)
	}
}
