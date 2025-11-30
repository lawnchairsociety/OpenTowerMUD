package wfc

import "testing"

func TestTileTypeString(t *testing.T) {
	tests := []struct {
		tt   TileType
		want string
	}{
		{TileEmpty, "empty"},
		{TileCorridor, "corridor"},
		{TileRoom, "room"},
		{TileDeadEnd, "dead_end"},
		{TileTreasure, "treasure"},
		{TileBoss, "boss"},
		{TileStairsUp, "stairs_up"},
		{TileStairsDown, "stairs_down"},
		{TileType(99), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.tt.String(); got != tc.want {
			t.Errorf("TileType(%d).String() = %q, want %q", tc.tt, got, tc.want)
		}
	}
}

func TestDirectionString(t *testing.T) {
	tests := []struct {
		d    Direction
		want string
	}{
		{North, "north"},
		{East, "east"},
		{South, "south"},
		{West, "west"},
		{Direction(99), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.d.String(); got != tc.want {
			t.Errorf("Direction(%d).String() = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestDirectionOpposite(t *testing.T) {
	tests := []struct {
		d    Direction
		want Direction
	}{
		{North, South},
		{South, North},
		{East, West},
		{West, East},
	}

	for _, tc := range tests {
		if got := tc.d.Opposite(); got != tc.want {
			t.Errorf("%s.Opposite() = %s, want %s", tc.d, got, tc.want)
		}
	}
}

func TestAllDirections(t *testing.T) {
	dirs := AllDirections()
	if len(dirs) != 4 {
		t.Errorf("AllDirections() returned %d directions, want 4", len(dirs))
	}

	expected := map[Direction]bool{North: false, East: false, South: false, West: false}
	for _, d := range dirs {
		if _, ok := expected[d]; !ok {
			t.Errorf("Unexpected direction: %s", d)
		}
		expected[d] = true
	}

	for d, found := range expected {
		if !found {
			t.Errorf("Missing direction: %s", d)
		}
	}
}

func TestNewTile(t *testing.T) {
	tile := NewTile(TileCorridor, 5, 10)

	if tile.Type != TileCorridor {
		t.Errorf("Type = %v, want %v", tile.Type, TileCorridor)
	}
	if tile.X != 5 {
		t.Errorf("X = %d, want 5", tile.X)
	}
	if tile.Y != 10 {
		t.Errorf("Y = %d, want 10", tile.Y)
	}
	if tile.Connections == nil {
		t.Error("Connections should not be nil")
	}
}

func TestTileConnections(t *testing.T) {
	tile := NewTile(TileRoom, 0, 0)

	// Initially no connections
	if tile.ConnectionCount() != 0 {
		t.Errorf("Initial ConnectionCount() = %d, want 0", tile.ConnectionCount())
	}

	// Add connections
	tile.SetConnection(North, true)
	tile.SetConnection(East, true)

	if tile.ConnectionCount() != 2 {
		t.Errorf("ConnectionCount() = %d, want 2", tile.ConnectionCount())
	}

	if !tile.HasConnection(North) {
		t.Error("HasConnection(North) should be true")
	}
	if !tile.HasConnection(East) {
		t.Error("HasConnection(East) should be true")
	}
	if tile.HasConnection(South) {
		t.Error("HasConnection(South) should be false")
	}
	if tile.HasConnection(West) {
		t.Error("HasConnection(West) should be false")
	}

	// Remove a connection
	tile.SetConnection(North, false)
	if tile.ConnectionCount() != 1 {
		t.Errorf("ConnectionCount() = %d, want 1", tile.ConnectionCount())
	}
}
