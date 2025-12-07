package wfc

import "testing"

func TestNewSolver(t *testing.T) {
	solver := NewSolver(10, 10, 42)

	if solver.Width != 10 {
		t.Errorf("Width = %d, want 10", solver.Width)
	}
	if solver.Height != 10 {
		t.Errorf("Height = %d, want 10", solver.Height)
	}
	if solver.Grid == nil {
		t.Fatal("Grid should not be nil")
	}
	if len(solver.Grid) != 10 {
		t.Errorf("Grid height = %d, want 10", len(solver.Grid))
	}
	if len(solver.Grid[0]) != 10 {
		t.Errorf("Grid width = %d, want 10", len(solver.Grid[0]))
	}
}

func TestCellEntropy(t *testing.T) {
	cell := &Cell{
		Possible: map[TileType]bool{
			TileCorridor:   true,
			TileRoom:       true,
			TileDeadEnd:    false,
			TileStairsUp:   true,
			TileStairsDown: false,
		},
	}

	if got := cell.Entropy(); got != 3 {
		t.Errorf("Entropy() = %d, want 3", got)
	}

	// All false
	cell.Possible = map[TileType]bool{
		TileCorridor: false,
		TileRoom:     false,
	}
	if got := cell.Entropy(); got != 0 {
		t.Errorf("Entropy() = %d, want 0", got)
	}
}

func TestSolverGeneratesRooms(t *testing.T) {
	solver := NewSolver(15, 15, 42)
	solver.MinRooms = 10
	solver.MaxRooms = 30
	solver.RequireStairs = false

	tiles, err := solver.Solve()
	if err != nil {
		t.Fatalf("Solve() failed: %v", err)
	}

	if len(tiles) < solver.MinRooms {
		t.Errorf("Too few tiles: %d < %d", len(tiles), solver.MinRooms)
	}

	if len(tiles) > solver.MaxRooms {
		t.Errorf("Too many tiles: %d > %d", len(tiles), solver.MaxRooms)
	}
}

func TestSolverProducesConnectedLayout(t *testing.T) {
	solver := NewSolver(15, 15, 123)
	solver.MinRooms = 15
	solver.MaxRooms = 30
	solver.RequireStairs = false

	tiles, err := solver.Solve()
	if err != nil {
		t.Fatalf("Solve() failed: %v", err)
	}

	if len(tiles) == 0 {
		t.Fatal("No tiles generated")
	}

	// Verify connectivity using BFS
	tileMap := make(map[string]*Tile)
	for _, tile := range tiles {
		tileMap[coordKey(tile.X, tile.Y)] = tile
	}

	visited := make(map[string]bool)
	queue := []*Tile{tiles[0]}
	visited[coordKey(tiles[0].X, tiles[0].Y)] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dir := range AllDirections() {
			if !current.HasConnection(dir) {
				continue
			}

			nx, ny := current.X, current.Y
			switch dir {
			case North:
				ny--
			case South:
				ny++
			case East:
				nx++
			case West:
				nx--
			}

			key := coordKey(nx, ny)
			if visited[key] {
				continue
			}

			if neighbor, ok := tileMap[key]; ok {
				visited[key] = true
				queue = append(queue, neighbor)
			}
		}
	}

	if len(visited) != len(tiles) {
		t.Errorf("Not all tiles connected: visited %d of %d", len(visited), len(tiles))
	}
}

func TestSolverSetRequireBoss(t *testing.T) {
	solver := NewSolver(8, 8, 456)

	// Initially boss is not required
	if solver.RequireBoss {
		t.Error("RequireBoss should default to false")
	}

	solver.SetRequireBoss(true)

	if !solver.RequireBoss {
		t.Error("RequireBoss should be true after SetRequireBoss(true)")
	}
}

func TestSolverGetNeighbor(t *testing.T) {
	solver := NewSolver(5, 5, 789)

	// Middle cell
	cell := solver.getNeighbor(2, 2, North)
	if cell == nil || cell.X != 2 || cell.Y != 1 {
		t.Error("North neighbor incorrect")
	}

	cell = solver.getNeighbor(2, 2, South)
	if cell == nil || cell.X != 2 || cell.Y != 3 {
		t.Error("South neighbor incorrect")
	}

	cell = solver.getNeighbor(2, 2, East)
	if cell == nil || cell.X != 3 || cell.Y != 2 {
		t.Error("East neighbor incorrect")
	}

	cell = solver.getNeighbor(2, 2, West)
	if cell == nil || cell.X != 1 || cell.Y != 2 {
		t.Error("West neighbor incorrect")
	}

	// Edge cases - out of bounds
	if solver.getNeighbor(0, 0, North) != nil {
		t.Error("North of (0,0) should be nil")
	}
	if solver.getNeighbor(0, 0, West) != nil {
		t.Error("West of (0,0) should be nil")
	}
	if solver.getNeighbor(4, 4, South) != nil {
		t.Error("South of (4,4) should be nil")
	}
	if solver.getNeighbor(4, 4, East) != nil {
		t.Error("East of (4,4) should be nil")
	}
}

func TestSolverNeighborCoords(t *testing.T) {
	solver := NewSolver(5, 5, 0)

	tests := []struct {
		x, y      int
		dir       Direction
		wantX, wantY int
	}{
		{2, 2, North, 2, 1},
		{2, 2, South, 2, 3},
		{2, 2, East, 3, 2},
		{2, 2, West, 1, 2},
	}

	for _, tc := range tests {
		gotX, gotY := solver.neighborCoords(tc.x, tc.y, tc.dir)
		if gotX != tc.wantX || gotY != tc.wantY {
			t.Errorf("neighborCoords(%d, %d, %s) = (%d, %d), want (%d, %d)",
				tc.x, tc.y, tc.dir, gotX, gotY, tc.wantX, tc.wantY)
		}
	}
}

func TestSortTilesByPosition(t *testing.T) {
	tiles := []*Tile{
		NewTile(TileCorridor, 3, 2),
		NewTile(TileCorridor, 1, 1),
		NewTile(TileCorridor, 2, 1),
		NewTile(TileCorridor, 0, 0),
	}

	SortTilesByPosition(tiles)

	expected := []struct{ x, y int }{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 2},
	}

	for i, exp := range expected {
		if tiles[i].X != exp.x || tiles[i].Y != exp.y {
			t.Errorf("tiles[%d] = (%d, %d), want (%d, %d)", i, tiles[i].X, tiles[i].Y, exp.x, exp.y)
		}
	}
}

func TestSolverCountConnections(t *testing.T) {
	solver := NewSolver(5, 5, 0)

	// Set up some connections
	cell := solver.Grid[2][2]
	cell.Connections[North] = true
	cell.Connections[East] = true

	if got := solver.countConnections(2, 2); got != 2 {
		t.Errorf("countConnections(2, 2) = %d, want 2", got)
	}

	// No connections
	if got := solver.countConnections(0, 0); got != 0 {
		t.Errorf("countConnections(0, 0) = %d, want 0", got)
	}
}

func TestCoordKey(t *testing.T) {
	tests := []struct {
		x, y int
		want string
	}{
		{0, 0, "0,0"},
		{5, 10, "5,10"},
		{-1, -2, "-1,-2"},
	}

	for _, tc := range tests {
		if got := coordKey(tc.x, tc.y); got != tc.want {
			t.Errorf("coordKey(%d, %d) = %q, want %q", tc.x, tc.y, got, tc.want)
		}
	}
}

// TestIsConnectedWithConnectedTiles tests the isConnected function with connected tiles
func TestIsConnectedWithConnectedTiles(t *testing.T) {
	solver := NewSolver(10, 10, 0)

	// Create a simple connected layout: 3 tiles in a row
	tiles := []*Tile{
		NewTile(TileCorridor, 0, 0),
		NewTile(TileCorridor, 1, 0),
		NewTile(TileCorridor, 2, 0),
	}
	// Connect them: 0 -> 1 -> 2
	tiles[0].SetConnection(East, true)
	tiles[1].SetConnection(West, true)
	tiles[1].SetConnection(East, true)
	tiles[2].SetConnection(West, true)

	if !solver.isConnected(tiles) {
		t.Error("Connected tiles should pass isConnected check")
	}
}

// TestIsConnectedWithDisconnectedTiles tests the isConnected function with disconnected tiles
func TestIsConnectedWithDisconnectedTiles(t *testing.T) {
	solver := NewSolver(10, 10, 0)

	// Create disconnected tiles: two separate groups
	tiles := []*Tile{
		NewTile(TileCorridor, 0, 0),
		NewTile(TileCorridor, 1, 0),
		NewTile(TileCorridor, 5, 5), // Disconnected
	}
	// Connect only first two
	tiles[0].SetConnection(East, true)
	tiles[1].SetConnection(West, true)
	// tiles[2] has no connections

	if solver.isConnected(tiles) {
		t.Error("Disconnected tiles should fail isConnected check")
	}
}

// TestIsConnectedEmptyTiles tests isConnected with empty tile list
func TestIsConnectedEmptyTiles(t *testing.T) {
	solver := NewSolver(10, 10, 0)

	if !solver.isConnected([]*Tile{}) {
		t.Error("Empty tile list should be considered connected")
	}
}

// TestIsConnectedSingleTile tests isConnected with a single tile
func TestIsConnectedSingleTile(t *testing.T) {
	solver := NewSolver(10, 10, 0)

	tiles := []*Tile{NewTile(TileCorridor, 5, 5)}

	if !solver.isConnected(tiles) {
		t.Error("Single tile should be considered connected")
	}
}

// TestConnectivityMultipleSeeds tests that WFC consistently produces connected layouts
func TestConnectivityMultipleSeeds(t *testing.T) {
	seeds := []int64{1, 42, 100, 255, 1000, 5000, 9999}

	for _, seed := range seeds {
		solver := NewSolver(12, 12, seed)
		solver.MinRooms = 15
		solver.MaxRooms = 30
		solver.RequireStairs = false

		tiles, err := solver.Solve()
		if err != nil {
			t.Fatalf("Seed %d: Solve() failed: %v", seed, err)
		}

		// Verify all tiles are connected
		if !solver.isConnected(tiles) {
			t.Errorf("Seed %d: Generated layout is not connected", seed)
		}

		// Verify BFS reaches all tiles
		if len(tiles) > 0 {
			visited := bfsTiles(tiles)
			if len(visited) != len(tiles) {
				t.Errorf("Seed %d: BFS visited %d tiles but generated %d tiles",
					seed, len(visited), len(tiles))
			}
		}
	}
}

// TestConnectivityWithBossAndStairs tests connectivity when boss and stairs are required
func TestConnectivityWithBossAndStairs(t *testing.T) {
	solver := NewSolver(15, 15, 12345)
	solver.MinRooms = 20
	solver.MaxRooms = 40
	solver.RequireStairs = true
	solver.SetRequireBoss(true)

	tiles, err := solver.Solve()
	if err != nil {
		t.Fatalf("Solve() failed: %v", err)
	}

	// Verify connectivity
	if !solver.isConnected(tiles) {
		t.Error("Layout with boss and stairs should be connected")
	}

	// Verify special tiles exist and are reachable
	var stairsUp, stairsDown, boss *Tile
	for _, tile := range tiles {
		switch tile.Type {
		case TileStairsUp:
			stairsUp = tile
		case TileStairsDown:
			stairsDown = tile
		case TileBoss:
			boss = tile
		}
	}

	// All special tiles should be reachable from any starting point
	visited := bfsTiles(tiles)
	if stairsUp != nil && !visited[coordKey(stairsUp.X, stairsUp.Y)] {
		t.Error("Stairs up tile is not reachable")
	}
	if stairsDown != nil && !visited[coordKey(stairsDown.X, stairsDown.Y)] {
		t.Error("Stairs down tile is not reachable")
	}
	if boss != nil && !visited[coordKey(boss.X, boss.Y)] {
		t.Error("Boss tile is not reachable")
	}
}

// bfsTiles performs BFS traversal on WFC tiles and returns visited coordinates
func bfsTiles(tiles []*Tile) map[string]bool {
	if len(tiles) == 0 {
		return map[string]bool{}
	}

	tileMap := make(map[string]*Tile)
	for _, t := range tiles {
		tileMap[coordKey(t.X, t.Y)] = t
	}

	visited := make(map[string]bool)
	queue := []*Tile{tiles[0]}
	visited[coordKey(tiles[0].X, tiles[0].Y)] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dir := range AllDirections() {
			if !current.HasConnection(dir) {
				continue
			}

			nx, ny := current.X, current.Y
			switch dir {
			case North:
				ny--
			case South:
				ny++
			case East:
				nx++
			case West:
				nx--
			}

			key := coordKey(nx, ny)
			if visited[key] {
				continue
			}

			if neighbor, ok := tileMap[key]; ok {
				visited[key] = true
				queue = append(queue, neighbor)
			}
		}
	}

	return visited
}
