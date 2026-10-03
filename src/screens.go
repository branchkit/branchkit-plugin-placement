package main

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/branchkit/plugin-sdk-go"
)

// screenOrder lists display indices left to right (top to bottom where
// two share a left edge): the numbering "screen 1", "screen 2" uses, the
// way the displays sit on the desk rather than the order the OS reports.
func screenOrder(displays []branchkit.DisplayInfo) []int {
	idx := make([]int, len(displays))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		da, db := displays[idx[a]], displays[idx[b]]
		if da.X != db.X {
			return da.X < db.X
		}
		return da.Y < db.Y
	})
	return idx
}

// targetScreen resolves "2" (left-to-right number) or a direction (left,
// right, up, down: the nearest display whose centre lies that way) from
// the display at index from.
func targetScreen(displays []branchkit.DisplayInfo, from int, spec string) (int, error) {
	if n, err := strconv.Atoi(spec); err == nil {
		order := screenOrder(displays)
		if n < 1 || n > len(order) {
			return 0, fmt.Errorf("there is no screen %d (this system has %d)", n, len(order))
		}
		return order[n-1], nil
	}
	cx := func(d branchkit.DisplayInfo) int { return d.X + d.W/2 }
	cy := func(d branchkit.DisplayInfo) int { return d.Y + d.H/2 }
	src := displays[from]
	best, bestDist := -1, 0
	for i, d := range displays {
		if i == from {
			continue
		}
		dx, dy := cx(d)-cx(src), cy(d)-cy(src)
		var along, across int
		switch spec {
		case "left":
			along, across = -dx, dy
		case "right":
			along, across = dx, dy
		case "up":
			along, across = -dy, dx
		case "down":
			along, across = dy, dx
		default:
			return 0, fmt.Errorf("%q is not a screen", spec)
		}
		// Mostly in that direction: further along it than across it.
		if along <= 0 || abs(across) > along {
			continue
		}
		dist := along*along + across*across
		if best < 0 || dist < bestDist {
			best, bestDist = i, dist
		}
	}
	if best < 0 {
		return 0, fmt.Errorf("there is no screen to the %s", directionWord(spec))
	}
	return best, nil
}

func directionWord(spec string) string {
	switch spec {
	case "up":
		return "top"
	case "down":
		return "bottom"
	}
	return spec
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (h *Host) handleToScreen(p ToScreenParams, req *branchkit.OnActionRequest) (any, error) {
	if p.Screen == "" {
		return nil, fmt.Errorf("screen: no screen given")
	}
	target := req.ActiveWindowID
	if p.App != nil && *p.App != "" {
		id, err := h.namedWindow(*p.App)
		if err != nil {
			return nil, fmt.Errorf("screen: %w", err)
		}
		target = &id
	}
	err := h.place(target, "screen", p.Screen,
		func(win *branchkit.WindowInfo, from int, displays []branchkit.DisplayInfo) (*branchkit.Rect, error) {
			if len(displays) < 2 {
				return nil, fmt.Errorf("there is only one display")
			}
			to, err := targetScreen(displays, from, p.Screen)
			if err != nil {
				return nil, err
			}
			if to == from {
				return nil, fmt.Errorf("the window is already there")
			}
			return mapBetween(win, usableArea(displays[from]), usableArea(displays[to])), nil
		})
	if err != nil {
		return nil, err
	}
	if p.App != nil && *p.App != "" {
		if err := h.plugin.NativeRaiseWindow(branchkit.NativeRaiseWindowRequest{WindowID: *target}); err != nil {
			branchkit.Logf("placement", "screen: raise %s: %v", *target, err)
		}
	}
	return nil, nil
}
