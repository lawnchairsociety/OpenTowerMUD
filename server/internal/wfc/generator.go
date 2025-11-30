package wfc

import (
	"fmt"
	"math/rand"
)

// FloorConfig contains parameters for floor generation
type FloorConfig struct {
	FloorNumber   int   // The floor number (1-indexed, 0 = city)
	TowerSeed     int64 // Base tower seed
	MinRooms      int   // Minimum number of rooms
	MaxRooms      int   // Maximum number of rooms
	TreasureCount int   // Number of treasure rooms to place
	IsBossFloor   bool  // Whether this is a boss floor (every 10th)
}

// DefaultFloorConfig returns reasonable defaults for a floor
func DefaultFloorConfig(floorNumber int, towerSeed int64) *FloorConfig {
	cfg := &FloorConfig{
		FloorNumber:   floorNumber,
		TowerSeed:     towerSeed,
		MinRooms:      20,
		MaxRooms:      50,
		TreasureCount: 1 + (floorNumber / 5), // More treasure on higher floors
		IsBossFloor:   floorNumber > 0 && floorNumber%10 == 0,
	}

	// Limit treasure count
	if cfg.TreasureCount > 3 {
		cfg.TreasureCount = 3
	}

	return cfg
}

// GeneratedFloor represents the output of floor generation
type GeneratedFloor struct {
	FloorNumber   int
	Tiles         []*Tile
	StairsUpTile  *Tile // The tile with stairs going up (nil for floor 0)
	StairsDownTile *Tile // The tile with stairs going down (nil for floor 1)
	BossTile      *Tile // The boss tile (nil if not a boss floor)
	TreasureTiles []*Tile
	Width, Height int
}

// Generator handles floor generation with constraints
type Generator struct {
	config     *FloorConfig
	rng        *rand.Rand
	maxRetries int
}

// NewGenerator creates a new floor generator
func NewGenerator(config *FloorConfig) *Generator {
	// Floor seed = tower seed + floor number for reproducibility
	floorSeed := config.TowerSeed + int64(config.FloorNumber)

	return &Generator{
		config:     config,
		rng:        rand.New(rand.NewSource(floorSeed)),
		maxRetries: 50,
	}
}

// Generate creates a floor layout
func (g *Generator) Generate() (*GeneratedFloor, error) {
	// Calculate grid size based on desired room count
	// Average ~30 rooms, so we need a grid that can support that
	// With ~40% fill rate, we need grid of ~75 cells for 30 rooms
	gridSize := g.calculateGridSize()

	var result *GeneratedFloor
	var lastErr error

	for attempt := 0; attempt < g.maxRetries; attempt++ {
		solver := NewSolver(gridSize, gridSize, g.config.TowerSeed+int64(g.config.FloorNumber)+int64(attempt*1000))
		solver.MinRooms = g.config.MinRooms
		solver.MaxRooms = g.config.MaxRooms
		solver.RequireStairs = g.config.FloorNumber > 0 // No stairs on ground floor (city)
		solver.SetRequireBoss(g.config.IsBossFloor)

		tiles, err := solver.Solve()
		if err != nil {
			lastErr = err
			continue
		}

		// Validate room count
		if len(tiles) < g.config.MinRooms {
			lastErr = fmt.Errorf("too few rooms: got %d, need %d", len(tiles), g.config.MinRooms)
			continue
		}
		if len(tiles) > g.config.MaxRooms {
			// Prune extra tiles if we have too many
			tiles = g.pruneTiles(tiles, g.config.MaxRooms)
		}

		// Post-process: ensure required tiles exist
		result = &GeneratedFloor{
			FloorNumber: g.config.FloorNumber,
			Tiles:       tiles,
			Width:       gridSize,
			Height:      gridSize,
		}

		// Find or place special tiles
		if err := g.placeSpecialTiles(result); err != nil {
			lastErr = err
			continue
		}

		return result, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", g.maxRetries, lastErr)
	}
	return nil, ErrNoSolution
}

// calculateGridSize determines an appropriate grid size for the floor
func (g *Generator) calculateGridSize() int {
	// Target room count is average of min/max
	target := (g.config.MinRooms + g.config.MaxRooms) / 2
	// Assume ~40% fill rate, so grid needs 2.5x the room count
	size := int(float64(target) * 2.5)
	if size < 8 {
		size = 8
	}
	if size > 15 {
		size = 15
	}
	return size
}

// pruneTiles removes tiles to get under the max count while maintaining connectivity
func (g *Generator) pruneTiles(tiles []*Tile, maxCount int) []*Tile {
	if len(tiles) <= maxCount {
		return tiles
	}

	// Find dead ends and low-connectivity tiles to remove
	// Keep removing until we're at max count
	for len(tiles) > maxCount {
		// Find a removable tile (dead end preferred)
		removeIdx := -1
		for i, t := range tiles {
			// Don't remove stairs, boss, or treasure
			if t.Type == TileStairsUp || t.Type == TileStairsDown ||
				t.Type == TileBoss || t.Type == TileTreasure {
				continue
			}
			if t.ConnectionCount() == 1 {
				removeIdx = i
				break
			}
		}

		if removeIdx == -1 {
			// No good candidates, stop pruning
			break
		}

		// Remove the tile
		tiles = append(tiles[:removeIdx], tiles[removeIdx+1:]...)
	}

	return tiles
}

// placeSpecialTiles ensures stairs, boss, and treasure rooms exist
func (g *Generator) placeSpecialTiles(floor *GeneratedFloor) error {
	tiles := floor.Tiles

	// Find existing special tiles
	var stairsUpTile, stairsDownTile, bossTile *Tile
	var treasureTiles []*Tile

	for _, t := range tiles {
		switch t.Type {
		case TileStairsUp:
			if stairsUpTile == nil {
				stairsUpTile = t
			}
		case TileStairsDown:
			if stairsDownTile == nil {
				stairsDownTile = t
			}
		case TileBoss:
			if bossTile == nil {
				bossTile = t
			}
		case TileTreasure:
			treasureTiles = append(treasureTiles, t)
		}
	}

	// Ensure stairs up exists (if not floor 0) - leads to next floor
	if g.config.FloorNumber > 0 && stairsUpTile == nil {
		stairsUpTile = g.convertToType(tiles, TileStairsUp, []TileType{TileDeadEnd, TileRoom, TileCorridor})
		if stairsUpTile == nil {
			return fmt.Errorf("failed to place stairs up room")
		}
	}

	// Ensure stairs down exists (if not floor 0) - comes from previous floor
	// Floor 1 connects down to the city tower entrance
	if g.config.FloorNumber > 0 && stairsDownTile == nil {
		stairsDownTile = g.convertToType(tiles, TileStairsDown, []TileType{TileDeadEnd, TileRoom, TileCorridor})
		if stairsDownTile == nil {
			return fmt.Errorf("failed to place stairs down room")
		}
	}

	// Ensure boss exists on boss floors
	if g.config.IsBossFloor && bossTile == nil {
		bossTile = g.convertToType(tiles, TileBoss, []TileType{TileDeadEnd, TileRoom})
		if bossTile == nil {
			return fmt.Errorf("failed to place boss room")
		}
	}

	// Ensure enough treasure rooms
	for len(treasureTiles) < g.config.TreasureCount {
		treasureTile := g.convertToType(tiles, TileTreasure, []TileType{TileDeadEnd, TileRoom})
		if treasureTile == nil {
			break // Can't place more treasures
		}
		treasureTiles = append(treasureTiles, treasureTile)
	}

	floor.StairsUpTile = stairsUpTile
	floor.StairsDownTile = stairsDownTile
	floor.BossTile = bossTile
	floor.TreasureTiles = treasureTiles

	return nil
}

// convertToType finds a tile of the preferred types and converts it
func (g *Generator) convertToType(tiles []*Tile, newType TileType, preferredTypes []TileType) *Tile {
	// First, try to find a tile of a preferred type
	for _, prefType := range preferredTypes {
		candidates := []*Tile{}
		for _, t := range tiles {
			if t.Type == prefType {
				candidates = append(candidates, t)
			}
		}
		if len(candidates) > 0 {
			chosen := candidates[g.rng.Intn(len(candidates))]
			chosen.Type = newType
			return chosen
		}
	}
	return nil
}

// GetRoomID generates a unique room ID for a tile on a floor
func GetRoomID(floorNumber, x, y int) string {
	return fmt.Sprintf("floor%d_%d_%d", floorNumber, x, y)
}
