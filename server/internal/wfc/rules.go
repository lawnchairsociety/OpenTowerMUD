package wfc

// Rules defines adjacency constraints for WFC
type Rules struct {
	// MinConnections is the minimum number of connections required for each tile type
	MinConnections map[TileType]int
	// MaxConnections is the maximum number of connections allowed for each tile type
	MaxConnections map[TileType]int
	// CanConnect defines whether two tile types can connect to each other
	// Key format: tile1 + tile2 (order doesn't matter)
	CanConnect map[TileType]map[TileType]bool
}

// DefaultRules returns the standard adjacency rules for tower generation
func DefaultRules() *Rules {
	r := &Rules{
		MinConnections: make(map[TileType]int),
		MaxConnections: make(map[TileType]int),
		CanConnect:     make(map[TileType]map[TileType]bool),
	}

	// Connection count constraints
	// Corridor: 2-4 connections (passageways)
	r.MinConnections[TileCorridor] = 2
	r.MaxConnections[TileCorridor] = 4

	// Room: 1-4 connections (flexible)
	r.MinConnections[TileRoom] = 1
	r.MaxConnections[TileRoom] = 4

	// Dead end: exactly 1 connection
	r.MinConnections[TileDeadEnd] = 1
	r.MaxConnections[TileDeadEnd] = 1

	// Stairs Up: exactly 1 connection (alcove with stairs going up)
	r.MinConnections[TileStairsUp] = 1
	r.MaxConnections[TileStairsUp] = 1

	// Stairs Down: exactly 1 connection (alcove with stairs coming down)
	r.MinConnections[TileStairsDown] = 1
	r.MaxConnections[TileStairsDown] = 1

	// Treasure: 1-2 connections (often at end of path)
	r.MinConnections[TileTreasure] = 1
	r.MaxConnections[TileTreasure] = 2

	// Boss: 1-2 connections (usually accessible from one direction)
	r.MinConnections[TileBoss] = 1
	r.MaxConnections[TileBoss] = 2

	// Initialize CanConnect maps
	allTypes := []TileType{TileCorridor, TileRoom, TileDeadEnd, TileTreasure, TileBoss, TileStairsUp, TileStairsDown}
	for _, t := range allTypes {
		r.CanConnect[t] = make(map[TileType]bool)
	}

	// Define what can connect to what
	// Corridors can connect to everything
	r.setCanConnect(TileCorridor, TileCorridor, true)
	r.setCanConnect(TileCorridor, TileRoom, true)
	r.setCanConnect(TileCorridor, TileDeadEnd, true)
	r.setCanConnect(TileCorridor, TileStairsUp, true)
	r.setCanConnect(TileCorridor, TileStairsDown, true)
	r.setCanConnect(TileCorridor, TileTreasure, true)
	r.setCanConnect(TileCorridor, TileBoss, true)

	// Rooms can connect to corridors, other rooms, and special rooms
	r.setCanConnect(TileRoom, TileRoom, true)
	r.setCanConnect(TileRoom, TileDeadEnd, true)
	r.setCanConnect(TileRoom, TileStairsUp, true)
	r.setCanConnect(TileRoom, TileStairsDown, true)
	r.setCanConnect(TileRoom, TileTreasure, true)
	r.setCanConnect(TileRoom, TileBoss, true)

	// Dead ends connect to corridors and rooms only
	r.setCanConnect(TileDeadEnd, TileDeadEnd, false) // Two dead ends can't connect

	// Stairs Up connect to corridors and rooms only
	r.setCanConnect(TileStairsUp, TileStairsUp, false)
	r.setCanConnect(TileStairsUp, TileStairsDown, false) // Up and down stairs don't connect directly
	r.setCanConnect(TileStairsUp, TileDeadEnd, false)
	r.setCanConnect(TileStairsUp, TileTreasure, false)
	r.setCanConnect(TileStairsUp, TileBoss, false)

	// Stairs Down connect to corridors and rooms only
	r.setCanConnect(TileStairsDown, TileStairsDown, false)
	r.setCanConnect(TileStairsDown, TileDeadEnd, false)
	r.setCanConnect(TileStairsDown, TileTreasure, false)
	r.setCanConnect(TileStairsDown, TileBoss, false)

	// Treasure rooms
	r.setCanConnect(TileTreasure, TileTreasure, false) // Treasures don't connect
	r.setCanConnect(TileTreasure, TileDeadEnd, false)
	r.setCanConnect(TileTreasure, TileBoss, false)

	// Boss rooms
	r.setCanConnect(TileBoss, TileBoss, false)
	r.setCanConnect(TileBoss, TileDeadEnd, false)

	return r
}

// setCanConnect sets bidirectional connection permission
func (r *Rules) setCanConnect(t1, t2 TileType, allowed bool) {
	if r.CanConnect[t1] == nil {
		r.CanConnect[t1] = make(map[TileType]bool)
	}
	if r.CanConnect[t2] == nil {
		r.CanConnect[t2] = make(map[TileType]bool)
	}
	r.CanConnect[t1][t2] = allowed
	r.CanConnect[t2][t1] = allowed
}

// CanTypesConnect returns true if two tile types can be adjacent
func (r *Rules) CanTypesConnect(t1, t2 TileType) bool {
	if t1 == TileEmpty || t2 == TileEmpty {
		return true // Empty tiles are compatible with everything
	}
	if r.CanConnect[t1] == nil {
		return false
	}
	return r.CanConnect[t1][t2]
}

// ValidConnectionCount returns true if the connection count is valid for the tile type
func (r *Rules) ValidConnectionCount(tileType TileType, count int) bool {
	if tileType == TileEmpty {
		return true
	}
	min, hasMin := r.MinConnections[tileType]
	max, hasMax := r.MaxConnections[tileType]

	if hasMin && count < min {
		return false
	}
	if hasMax && count > max {
		return false
	}
	return true
}

// GetMinConnections returns the minimum connections for a tile type
func (r *Rules) GetMinConnections(tileType TileType) int {
	if min, ok := r.MinConnections[tileType]; ok {
		return min
	}
	return 1 // Default minimum
}

// GetMaxConnections returns the maximum connections for a tile type
func (r *Rules) GetMaxConnections(tileType TileType) int {
	if max, ok := r.MaxConnections[tileType]; ok {
		return max
	}
	return 4 // Default maximum
}
