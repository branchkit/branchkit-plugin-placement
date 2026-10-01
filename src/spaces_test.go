package main

import (
	"strings"
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// Desktop N is the Nth USER space in list_spaces order. Fullscreen spaces
// sit between them on macOS and must not shift the count; on X11 the ids
// are positions, on Windows folded GUIDs, so the id is never the number.
func TestDeskSpaceIDCountsOnlyUserSpacesInOrder(t *testing.T) {
	spaces := []branchkit.SpaceInfo{
		{SpaceID: 901, SpaceType: "user", DisplayID: 1},
		{SpaceID: 455, SpaceType: "fullscreen", DisplayID: 1},
		{SpaceID: 7, SpaceType: "user", DisplayID: 1},
		{SpaceID: 1234, SpaceType: "user", DisplayID: 2},
	}
	for desk, want := range map[int]int{1: 901, 2: 7, 3: 1234} {
		got, err := deskSpaceID(spaces, desk)
		if err != nil || got != want {
			t.Errorf("desk %d: got (%d, %v), want %d", desk, got, err, want)
		}
	}
}

func TestDeskSpaceIDNamesTheDesktopsThatExist(t *testing.T) {
	spaces := []branchkit.SpaceInfo{
		{SpaceID: 1, SpaceType: "user"},
		{SpaceID: 2, SpaceType: "user"},
	}
	_, err := deskSpaceID(spaces, 5)
	if err == nil || !strings.Contains(err.Error(), "no desktop 5") || !strings.Contains(err.Error(), "has 2") {
		t.Fatalf("err = %v, want it to say desktop 5 does not exist and there are 2", err)
	}
}
