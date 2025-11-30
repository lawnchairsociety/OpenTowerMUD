package wfc

import "testing"

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()

	if rules == nil {
		t.Fatal("DefaultRules() returned nil")
	}

	// Test min/max connections exist for all types
	types := []TileType{TileCorridor, TileRoom, TileDeadEnd, TileTreasure, TileBoss, TileStairsUp, TileStairsDown}
	for _, tt := range types {
		if _, ok := rules.MinConnections[tt]; !ok {
			t.Errorf("Missing MinConnections for %s", tt)
		}
		if _, ok := rules.MaxConnections[tt]; !ok {
			t.Errorf("Missing MaxConnections for %s", tt)
		}
	}
}

func TestRulesConnectionCounts(t *testing.T) {
	rules := DefaultRules()

	tests := []struct {
		tileType TileType
		minConn  int
		maxConn  int
	}{
		{TileCorridor, 2, 4},
		{TileRoom, 1, 4},
		{TileDeadEnd, 1, 1},
		{TileStairsUp, 1, 1},   // Stairs are like dead-ends (alcove feel)
		{TileStairsDown, 1, 1}, // Stairs are like dead-ends (alcove feel)
		{TileTreasure, 1, 2},
		{TileBoss, 1, 2},
	}

	for _, tc := range tests {
		if got := rules.GetMinConnections(tc.tileType); got != tc.minConn {
			t.Errorf("GetMinConnections(%s) = %d, want %d", tc.tileType, got, tc.minConn)
		}
		if got := rules.GetMaxConnections(tc.tileType); got != tc.maxConn {
			t.Errorf("GetMaxConnections(%s) = %d, want %d", tc.tileType, got, tc.maxConn)
		}
	}
}

func TestRulesValidConnectionCount(t *testing.T) {
	rules := DefaultRules()

	tests := []struct {
		tileType TileType
		count    int
		valid    bool
	}{
		{TileCorridor, 1, false}, // Below min
		{TileCorridor, 2, true},  // At min
		{TileCorridor, 3, true},  // Middle
		{TileCorridor, 4, true},  // At max
		{TileCorridor, 5, false}, // Above max

		{TileDeadEnd, 0, false}, // Below min
		{TileDeadEnd, 1, true},  // Exactly 1
		{TileDeadEnd, 2, false}, // Above max

		{TileEmpty, 0, true},  // Empty always valid
		{TileEmpty, 10, true}, // Empty always valid
	}

	for _, tc := range tests {
		if got := rules.ValidConnectionCount(tc.tileType, tc.count); got != tc.valid {
			t.Errorf("ValidConnectionCount(%s, %d) = %v, want %v", tc.tileType, tc.count, got, tc.valid)
		}
	}
}

func TestRulesCanTypesConnect(t *testing.T) {
	rules := DefaultRules()

	tests := []struct {
		t1, t2 TileType
		can    bool
	}{
		// Corridors connect to everything
		{TileCorridor, TileCorridor, true},
		{TileCorridor, TileRoom, true},
		{TileCorridor, TileStairsUp, true},
		{TileCorridor, TileStairsDown, true},
		{TileCorridor, TileTreasure, true},
		{TileCorridor, TileBoss, true},
		{TileCorridor, TileDeadEnd, true},

		// Rooms connect to most things
		{TileRoom, TileRoom, true},
		{TileRoom, TileStairsUp, true},
		{TileRoom, TileStairsDown, true},
		{TileRoom, TileTreasure, true},

		// Dead ends don't connect to each other
		{TileDeadEnd, TileDeadEnd, false},

		// Stairs don't connect to each other
		{TileStairsUp, TileStairsUp, false},
		{TileStairsDown, TileStairsDown, false},
		{TileStairsUp, TileStairsDown, false},

		// Treasure rooms don't connect to each other
		{TileTreasure, TileTreasure, false},

		// Boss rooms don't connect to each other
		{TileBoss, TileBoss, false},

		// Empty is always compatible
		{TileEmpty, TileCorridor, true},
		{TileEmpty, TileEmpty, true},
	}

	for _, tc := range tests {
		if got := rules.CanTypesConnect(tc.t1, tc.t2); got != tc.can {
			t.Errorf("CanTypesConnect(%s, %s) = %v, want %v", tc.t1, tc.t2, got, tc.can)
		}
		// Test symmetry
		if got := rules.CanTypesConnect(tc.t2, tc.t1); got != tc.can {
			t.Errorf("CanTypesConnect(%s, %s) = %v, want %v (symmetry)", tc.t2, tc.t1, got, tc.can)
		}
	}
}
