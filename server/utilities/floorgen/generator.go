package main

import (
	"fmt"
	"path/filepath"

	"github.com/lawnchairsociety/opentowermud/server/internal/wfc"
)

// FloorGenerator handles generating floors and writing them to YAML
type FloorGenerator struct {
	TowerID   string
	Seed      int64
	OutputDir string
}

// NewFloorGenerator creates a new floor generator
func NewFloorGenerator(towerID string, seed int64, outputDir string) *FloorGenerator {
	return &FloorGenerator{
		TowerID:   towerID,
		Seed:      seed,
		OutputDir: outputDir,
	}
}

// GenerateFloor generates a single floor and writes it to YAML
func (g *FloorGenerator) GenerateFloor(floorNum int) error {
	// Create WFC config for this floor
	config := wfc.DefaultFloorConfig(floorNum, g.Seed)

	// Generate the floor using WFC
	gen := wfc.NewGenerator(config)
	generated, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("WFC generation failed: %w", err)
	}

	// Convert to YAML format
	floorYAML := g.convertToYAML(floorNum, generated)

	// Write to file
	filename := fmt.Sprintf("floor_%d.yaml", floorNum)
	path := filepath.Join(g.OutputDir, filename)

	if err := WriteFloorYAML(floorYAML, path); err != nil {
		return fmt.Errorf("failed to write YAML: %w", err)
	}

	return nil
}

// convertToYAML converts WFC output to our YAML format
func (g *FloorGenerator) convertToYAML(floorNum int, generated *wfc.GeneratedFloor) *FloorYAML {
	floor := &FloorYAML{
		Floor:         floorNum,
		Tower:         g.TowerID,
		GeneratedSeed: g.Seed + int64(floorNum), // Same formula as WFC uses
		Rooms:         make(map[string]*RoomYAML),
	}

	// Build a map of tile positions for exit lookups
	tileMap := make(map[string]*wfc.Tile)
	for _, tile := range generated.Tiles {
		key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
		tileMap[key] = tile
	}

	// Convert tiles to rooms
	for _, tile := range generated.Tiles {
		roomID := g.getRoomID(floorNum, tile.X, tile.Y)
		room := &RoomYAML{
			Name:        generateRoomName(tile.Type, floorNum),
			Description: generateRoomDescription(tile.Type),
			Type:        tile.Type.String(),
			Features:    g.getFeaturesForTile(tile),
			Exits:       make(map[string]string),
		}

		// Build exits
		for _, dir := range wfc.AllDirections() {
			if !tile.HasConnection(dir) {
				continue
			}

			nx, ny := tile.X, tile.Y
			switch dir {
			case wfc.North:
				ny--
			case wfc.South:
				ny++
			case wfc.East:
				nx++
			case wfc.West:
				nx--
			}

			neighborKey := fmt.Sprintf("%d,%d", nx, ny)
			if _, ok := tileMap[neighborKey]; ok {
				neighborID := g.getRoomID(floorNum, nx, ny)
				room.Exits[dir.String()] = neighborID
			}
		}

		floor.Rooms[roomID] = room
	}

	// Set special room IDs
	if generated.StairsUpTile != nil {
		floor.StairsUp = g.getRoomID(floorNum, generated.StairsUpTile.X, generated.StairsUpTile.Y)
	}
	if generated.StairsDownTile != nil {
		floor.StairsDown = g.getRoomID(floorNum, generated.StairsDownTile.X, generated.StairsDownTile.Y)
		floor.PortalRoom = floor.StairsDown // Portal is at stairs down (entry point)
	}

	return floor
}

// getRoomID generates a unique room ID
func (g *FloorGenerator) getRoomID(floorNum, x, y int) string {
	return fmt.Sprintf("%s_f%d_r%d_%d", g.TowerID, floorNum, x, y)
}

// getFeaturesForTile returns the features list for a tile type
func (g *FloorGenerator) getFeaturesForTile(tile *wfc.Tile) []string {
	var features []string

	switch tile.Type {
	case wfc.TileStairsUp:
		features = append(features, "stairs_up")
	case wfc.TileStairsDown:
		features = append(features, "stairs_down", "portal")
	case wfc.TileTreasure:
		features = append(features, "treasure")
	case wfc.TileBoss:
		features = append(features, "boss")
	}

	return features
}

// generateRoomName creates a name for a room based on its type
func generateRoomName(tt wfc.TileType, floor int) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Tower Corridor (Floor %d)", floor)
	case wfc.TileRoom:
		return fmt.Sprintf("Tower Chamber (Floor %d)", floor)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Dead End (Floor %d)", floor)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Ascending Stairway (Floor %d)", floor)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Descending Stairway (Floor %d)", floor)
	case wfc.TileTreasure:
		return fmt.Sprintf("Treasure Room (Floor %d)", floor)
	case wfc.TileBoss:
		return fmt.Sprintf("Boss Chamber (Floor %d)", floor)
	default:
		return fmt.Sprintf("Unknown Room (Floor %d)", floor)
	}
}

// generateRoomDescription creates a description for a room based on its type
func generateRoomDescription(tt wfc.TileType) string {
	switch tt {
	case wfc.TileCorridor:
		return "A narrow stone corridor stretches before you. Torches flicker on the walls, casting dancing shadows."
	case wfc.TileRoom:
		return "You stand in a chamber within the tower. The ancient stone walls are cold to the touch."
	case wfc.TileDeadEnd:
		return "The passage ends here in a small alcove. Dust motes drift in the dim light."
	case wfc.TileStairsUp:
		return "A spiral staircase ascends into the darkness above. The stone steps are worn smooth by countless travelers."
	case wfc.TileStairsDown:
		return "A spiral staircase descends from above. A shimmering portal offers quick travel to floors you've visited."
	case wfc.TileTreasure:
		return "This chamber holds the remnants of some forgotten hoard. Glittering objects catch the torchlight."
	case wfc.TileBoss:
		return "An ominous presence fills this grand chamber. The air is thick with danger."
	default:
		return "You are in a room within the tower."
	}
}
