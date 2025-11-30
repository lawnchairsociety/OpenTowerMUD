package wfc

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
)

var (
	ErrContradiction = errors.New("wfc: contradiction - no valid tiles for cell")
	ErrMaxIterations = errors.New("wfc: exceeded maximum iterations")
	ErrInvalidSize   = errors.New("wfc: invalid grid size")
	ErrNoSolution    = errors.New("wfc: failed to find valid solution")
	ErrNotConnected  = errors.New("wfc: generated layout is not fully connected")
)

// Cell represents a single cell in the WFC grid during solving
type Cell struct {
	X, Y        int
	Possible    map[TileType]bool // Which tile types are still possible
	Collapsed   bool              // Whether this cell has been assigned
	Type        TileType          // The assigned type (if collapsed)
	Connections map[Direction]bool
}

// Entropy returns the number of possible states
func (c *Cell) Entropy() int {
	count := 0
	for _, possible := range c.Possible {
		if possible {
			count++
		}
	}
	return count
}

// Solver implements the Wave Function Collapse algorithm for dungeon generation
type Solver struct {
	Width, Height int
	Grid          [][]*Cell
	Rules         *Rules
	rng           *rand.Rand

	// Constraints
	MinRooms      int
	MaxRooms      int
	RequireStairs bool
	RequireBoss   bool

	// Active cells that can still grow
	frontier []struct{ x, y int }
}

// NewSolver creates a new WFC solver with the given dimensions
func NewSolver(width, height int, seed int64) *Solver {
	s := &Solver{
		Width:         width,
		Height:        height,
		Rules:         DefaultRules(),
		rng:           rand.New(rand.NewSource(seed)),
		MinRooms:      20,
		MaxRooms:      50,
		RequireStairs: true,
		RequireBoss:   false,
		frontier:      make([]struct{ x, y int }, 0),
	}

	s.initializeGrid()
	return s
}

// initializeGrid sets up an empty grid
func (s *Solver) initializeGrid() {
	s.Grid = make([][]*Cell, s.Height)
	for y := 0; y < s.Height; y++ {
		s.Grid[y] = make([]*Cell, s.Width)
		for x := 0; x < s.Width; x++ {
			s.Grid[y][x] = &Cell{
				X:           x,
				Y:           y,
				Possible:    make(map[TileType]bool),
				Collapsed:   false,
				Type:        TileEmpty,
				Connections: make(map[Direction]bool),
			}
		}
	}
}

// SetRequireBoss sets whether a boss room should be placed
func (s *Solver) SetRequireBoss(require bool) {
	s.RequireBoss = require
}

// Solve runs the WFC algorithm and returns the resulting grid
func (s *Solver) Solve() ([]*Tile, error) {
	// Start from the center
	startX := s.Width / 2
	startY := s.Height / 2

	// Place the first tile
	if err := s.placeInitialTile(startX, startY); err != nil {
		return nil, err
	}

	// Grow the dungeon
	tileCount := 1
	maxIterations := s.Width * s.Height * 10

	for i := 0; i < maxIterations && len(s.frontier) > 0; i++ {
		// Stop if we've reached max rooms
		if tileCount >= s.MaxRooms {
			break
		}

		// Pick a random frontier cell to expand from
		idx := s.rng.Intn(len(s.frontier))
		front := s.frontier[idx]

		// Try to expand in a random direction
		expanded := s.tryExpand(front.x, front.y)
		if expanded {
			tileCount++
		}

		// Remove from frontier if no more expansion possible
		if !s.canExpand(front.x, front.y) {
			s.frontier = append(s.frontier[:idx], s.frontier[idx+1:]...)
		}
	}

	// Validate room count
	if tileCount < s.MinRooms {
		return nil, fmt.Errorf("only generated %d rooms, need at least %d", tileCount, s.MinRooms)
	}

	// Extract tiles
	tiles := s.extractTiles()

	// Verify connectivity
	if !s.isConnected(tiles) {
		return nil, ErrNotConnected
	}

	return tiles, nil
}

// placeInitialTile places the first tile in the dungeon
func (s *Solver) placeInitialTile(x, y int) error {
	cell := s.Grid[y][x]
	cell.Type = TileCorridor // Start with a corridor
	cell.Collapsed = true

	// Add to frontier
	s.frontier = append(s.frontier, struct{ x, y int }{x, y})

	return nil
}

// tryExpand attempts to expand the dungeon from the given cell
func (s *Solver) tryExpand(x, y int) bool {
	cell := s.Grid[y][x]
	if !cell.Collapsed {
		return false
	}

	// Get available directions (unoccupied neighbors within bounds)
	var available []Direction
	for _, dir := range AllDirections() {
		nx, ny := s.neighborCoords(x, y, dir)
		if nx < 0 || nx >= s.Width || ny < 0 || ny >= s.Height {
			continue
		}
		neighbor := s.Grid[ny][nx]
		if !neighbor.Collapsed {
			available = append(available, dir)
		}
	}

	if len(available) == 0 {
		return false
	}

	// Check if we can still add connections based on cell type
	currentConnections := s.countConnections(x, y)
	maxConn := s.Rules.GetMaxConnections(cell.Type)
	if currentConnections >= maxConn {
		return false
	}

	// Pick a random direction
	dir := available[s.rng.Intn(len(available))]
	nx, ny := s.neighborCoords(x, y, dir)

	// Choose a tile type for the new cell
	newType := s.chooseTileType(nx, ny, dir.Opposite())
	if newType == TileEmpty {
		return false
	}

	// Place the new tile
	neighbor := s.Grid[ny][nx]
	neighbor.Type = newType
	neighbor.Collapsed = true
	neighbor.Connections[dir.Opposite()] = true
	cell.Connections[dir] = true

	// Add to frontier if it can expand further
	s.frontier = append(s.frontier, struct{ x, y int }{nx, ny})

	return true
}

// chooseTileType selects an appropriate tile type for a new cell
func (s *Solver) chooseTileType(x, y int, fromDir Direction) TileType {
	// Count existing neighbors to understand context
	existingNeighbors := 0
	for _, dir := range AllDirections() {
		nx, ny := s.neighborCoords(x, y, dir)
		if nx >= 0 && nx < s.Width && ny >= 0 && ny < s.Height {
			if s.Grid[ny][nx].Collapsed {
				existingNeighbors++
			}
		}
	}

	// Build weighted options
	type option struct {
		tileType TileType
		weight   int
	}
	var options []option

	// Corridor - good for connections (weight higher in middle of dungeon)
	options = append(options, option{TileCorridor, 5})

	// Room - general purpose
	options = append(options, option{TileRoom, 4})

	// Dead end - only if we're approaching max rooms or edge
	if existingNeighbors == 1 && s.rng.Float32() < 0.3 {
		options = append(options, option{TileDeadEnd, 2})
	}

	// Stairs up - rare, one per floor
	if s.RequireStairs && s.rng.Float32() < 0.05 {
		options = append(options, option{TileStairsUp, 1})
	}

	// Stairs down - rare, one per floor
	if s.RequireStairs && s.rng.Float32() < 0.05 {
		options = append(options, option{TileStairsDown, 1})
	}

	// Treasure - rare
	if s.rng.Float32() < 0.08 {
		options = append(options, option{TileTreasure, 1})
	}

	// Boss - very rare, only if required
	if s.RequireBoss && s.rng.Float32() < 0.03 {
		options = append(options, option{TileBoss, 1})
	}

	// Build weighted list
	var weighted []TileType
	for _, opt := range options {
		for i := 0; i < opt.weight; i++ {
			weighted = append(weighted, opt.tileType)
		}
	}

	if len(weighted) == 0 {
		return TileEmpty
	}

	return weighted[s.rng.Intn(len(weighted))]
}

// canExpand returns true if the cell can still expand to neighbors
func (s *Solver) canExpand(x, y int) bool {
	cell := s.Grid[y][x]
	if !cell.Collapsed {
		return false
	}

	// Check connection limits
	currentConnections := s.countConnections(x, y)
	maxConn := s.Rules.GetMaxConnections(cell.Type)
	if currentConnections >= maxConn {
		return false
	}

	// Check for available neighbors
	for _, dir := range AllDirections() {
		nx, ny := s.neighborCoords(x, y, dir)
		if nx >= 0 && nx < s.Width && ny >= 0 && ny < s.Height {
			if !s.Grid[ny][nx].Collapsed {
				return true
			}
		}
	}

	return false
}

// countConnections counts the number of connections for a cell
func (s *Solver) countConnections(x, y int) int {
	count := 0
	for _, dir := range AllDirections() {
		if s.Grid[y][x].Connections[dir] {
			count++
		}
	}
	return count
}

// neighborCoords returns the coordinates of a neighbor in the given direction
func (s *Solver) neighborCoords(x, y int, dir Direction) (int, int) {
	switch dir {
	case North:
		return x, y - 1
	case South:
		return x, y + 1
	case East:
		return x + 1, y
	case West:
		return x - 1, y
	}
	return x, y
}

// getNeighbor returns the neighbor cell in the given direction
func (s *Solver) getNeighbor(x, y int, dir Direction) *Cell {
	nx, ny := s.neighborCoords(x, y, dir)
	if nx < 0 || nx >= s.Width || ny < 0 || ny >= s.Height {
		return nil
	}
	return s.Grid[ny][nx]
}

// extractTiles converts the collapsed grid into Tile objects
func (s *Solver) extractTiles() []*Tile {
	var tiles []*Tile

	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			cell := s.Grid[y][x]
			if !cell.Collapsed || cell.Type == TileEmpty {
				continue
			}

			tile := NewTile(cell.Type, x, y)

			// Copy connections
			for dir, connected := range cell.Connections {
				tile.Connections[dir] = connected
			}

			tiles = append(tiles, tile)
		}
	}

	return tiles
}

// isConnected verifies that all tiles are reachable from any starting point
func (s *Solver) isConnected(tiles []*Tile) bool {
	if len(tiles) == 0 {
		return true
	}

	// Create a map for quick lookup
	tileMap := make(map[string]*Tile)
	for _, t := range tiles {
		key := coordKey(t.X, t.Y)
		tileMap[key] = t
	}

	// BFS from the first tile
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

	return len(visited) == len(tiles)
}

func coordKey(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

// SortTilesByPosition sorts tiles by Y then X for deterministic output
func SortTilesByPosition(tiles []*Tile) {
	sort.Slice(tiles, func(i, j int) bool {
		if tiles[i].Y != tiles[j].Y {
			return tiles[i].Y < tiles[j].Y
		}
		return tiles[i].X < tiles[j].X
	})
}
