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
			Name:        generateRoomName(g.TowerID, tile.Type, floorNum),
			Description: generateRoomDescription(g.TowerID, tile.Type, floorNum),
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

// getFloorTier returns the floor tier (1-5) based on floor number for 25-floor towers
// This matches the website lore floor groupings
func getFloorTier(floor int) int {
	switch {
	case floor <= 5:
		return 1
	case floor <= 10:
		return 2
	case floor <= 15:
		return 3
	case floor <= 20:
		return 4
	case floor <= 24:
		return 5
	default:
		return 6 // Boss floor (25)
	}
}

// getUnifiedFloorTier returns the floor tier for 100-floor unified tower
func getUnifiedFloorTier(floor int) int {
	switch {
	case floor <= 10:
		return 1 // Mirror Halls
	case floor <= 25:
		return 2 // Crucible of Races
	case floor <= 50:
		return 3 // Labyrinth of Lies
	case floor <= 75:
		return 4 // Gauntlet of Gods
	case floor <= 99:
		return 5 // The Threshold
	default:
		return 6 // The Summit (100)
	}
}

// generateRoomName creates a name for a room based on tower theme and tile type
func generateRoomName(tower string, tt wfc.TileType, floor int) string {
	// Dwarf tower uses "Level" instead of "Floor" and descending terminology
	floorLabel := fmt.Sprintf("Floor %d", floor)
	if tower == "dwarf" {
		floorLabel = fmt.Sprintf("Level %d", floor)
	} else if tower == "unified" {
		floorLabel = fmt.Sprintf("Spire %d", floor)
	}

	switch tower {
	case "human":
		return generateHumanRoomName(tt, floor, floorLabel)
	case "elf":
		return generateElfRoomName(tt, floor, floorLabel)
	case "dwarf":
		return generateDwarfRoomName(tt, floor, floorLabel)
	case "gnome":
		return generateGnomeRoomName(tt, floor, floorLabel)
	case "orc":
		return generateOrcRoomName(tt, floor, floorLabel)
	case "unified":
		return generateUnifiedRoomName(tt, floor, floorLabel)
	default:
		return generateDefaultRoomName(tt, floorLabel)
	}
}

func generateHumanRoomName(tt wfc.TileType, floor int, floorLabel string) string {
	tier := getFloorTier(floor)
	// Human tower: Shattered Atrium (1-5), Burning Library (6-10), Impossible Gallery (11-15),
	// Whispering Archives (16-20), Void Chambers (21-24), The Archive (25)
	var zoneName string
	switch tier {
	case 1:
		zoneName = "Shattered Atrium"
	case 2:
		zoneName = "Burning Library"
	case 3:
		zoneName = "Impossible Gallery"
	case 4:
		zoneName = "Whispering Archives"
	case 5:
		zoneName = "Void Chambers"
	default:
		zoneName = "The Archive"
	}

	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("%s Passage (%s)", zoneName, floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("%s Chamber (%s)", zoneName, floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("%s Alcove (%s)", zoneName, floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("%s Ascent (%s)", zoneName, floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("%s Descent (%s)", zoneName, floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("%s Treasury (%s)", zoneName, floorLabel)
	case wfc.TileBoss:
		if tier == 6 {
			return fmt.Sprintf("The Archive (%s)", floorLabel)
		}
		return fmt.Sprintf("%s Guardian Chamber (%s)", zoneName, floorLabel)
	default:
		return fmt.Sprintf("%s (%s)", zoneName, floorLabel)
	}
}

func generateElfRoomName(tt wfc.TileType, floor int, floorLabel string) string {
	tier := getFloorTier(floor)
	// Elf tower: Rotting Roots (1-5), Hollow Trunk (6-10), Canker Heart (11-15),
	// Twisted Branches (16-20), Crown of Thorns (21-24), Heart Chamber (25)
	var zoneName string
	switch tier {
	case 1:
		zoneName = "Rotting Roots"
	case 2:
		zoneName = "Hollow Trunk"
	case 3:
		zoneName = "Canker Heart"
	case 4:
		zoneName = "Twisted Branches"
	case 5:
		zoneName = "Crown of Thorns"
	default:
		zoneName = "Heart Chamber"
	}

	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("%s Tunnel (%s)", zoneName, floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("%s Hollow (%s)", zoneName, floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("%s Alcove (%s)", zoneName, floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("%s Ascent (%s)", zoneName, floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("%s Descent (%s)", zoneName, floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("%s Shrine (%s)", zoneName, floorLabel)
	case wfc.TileBoss:
		if tier == 6 {
			return fmt.Sprintf("The Heart Chamber (%s)", floorLabel)
		}
		return fmt.Sprintf("%s Guardian Hollow (%s)", zoneName, floorLabel)
	default:
		return fmt.Sprintf("%s (%s)", zoneName, floorLabel)
	}
}

func generateDwarfRoomName(tt wfc.TileType, floor int, floorLabel string) string {
	tier := getFloorTier(floor)
	// Dwarf tower: Sealed Shafts (1-5), Flooded Galleries (6-10), Mithril Veins (11-15),
	// The Collapse (16-20), The Breach (21-24), The Deep (25)
	var zoneName string
	switch tier {
	case 1:
		zoneName = "Sealed Shafts"
	case 2:
		zoneName = "Flooded Galleries"
	case 3:
		zoneName = "Mithril Veins"
	case 4:
		zoneName = "The Collapse"
	case 5:
		zoneName = "The Breach"
	default:
		zoneName = "The Deep"
	}

	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("%s Tunnel (%s)", zoneName, floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("%s Cavern (%s)", zoneName, floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("%s Dead End (%s)", zoneName, floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("%s Shaft Up (%s)", zoneName, floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("%s Shaft Down (%s)", zoneName, floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("%s Ore Chamber (%s)", zoneName, floorLabel)
	case wfc.TileBoss:
		if tier == 6 {
			return fmt.Sprintf("The Deep Guardian's Lair (%s)", floorLabel)
		}
		return fmt.Sprintf("%s Guardian Cavern (%s)", zoneName, floorLabel)
	default:
		return fmt.Sprintf("%s (%s)", zoneName, floorLabel)
	}
}

func generateGnomeRoomName(tt wfc.TileType, floor int, floorLabel string) string {
	tier := getFloorTier(floor)
	// Gnome tower: Assembly Lines (1-5), Steam Works (6-10), Calculation Engines (11-15),
	// Prototype Labs (16-20), Master Forge (21-24), The Core (25)
	var zoneName string
	switch tier {
	case 1:
		zoneName = "Assembly Lines"
	case 2:
		zoneName = "Steam Works"
	case 3:
		zoneName = "Calculation Engines"
	case 4:
		zoneName = "Prototype Labs"
	case 5:
		zoneName = "Master Forge"
	default:
		zoneName = "The Core"
	}

	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("%s Corridor (%s)", zoneName, floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("%s Chamber (%s)", zoneName, floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("%s Service Bay (%s)", zoneName, floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("%s Elevator Up (%s)", zoneName, floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("%s Elevator Down (%s)", zoneName, floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("%s Storage (%s)", zoneName, floorLabel)
	case wfc.TileBoss:
		if tier == 6 {
			return fmt.Sprintf("The Core (%s)", floorLabel)
		}
		return fmt.Sprintf("%s Control Room (%s)", zoneName, floorLabel)
	default:
		return fmt.Sprintf("%s (%s)", zoneName, floorLabel)
	}
}

func generateOrcRoomName(tt wfc.TileType, floor int, floorLabel string) string {
	tier := getFloorTier(floor)
	// Orc tower: Ossuary (1-5), Champions' Rest (6-10), Proving Grounds (11-15),
	// Hall of Chieftains (16-20), Beast's Spine (21-24), Skull Throne (25)
	var zoneName string
	switch tier {
	case 1:
		zoneName = "Ossuary"
	case 2:
		zoneName = "Champions' Rest"
	case 3:
		zoneName = "Proving Grounds"
	case 4:
		zoneName = "Hall of Chieftains"
	case 5:
		zoneName = "Beast's Spine"
	default:
		zoneName = "Skull Throne"
	}

	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("%s Passage (%s)", zoneName, floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("%s Chamber (%s)", zoneName, floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("%s Alcove (%s)", zoneName, floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("%s Ascent (%s)", zoneName, floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("%s Descent (%s)", zoneName, floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("%s Trophy Chamber (%s)", zoneName, floorLabel)
	case wfc.TileBoss:
		if tier == 6 {
			return fmt.Sprintf("The Skull Throne (%s)", floorLabel)
		}
		return fmt.Sprintf("%s Arena (%s)", zoneName, floorLabel)
	default:
		return fmt.Sprintf("%s (%s)", zoneName, floorLabel)
	}
}

func generateUnifiedRoomName(tt wfc.TileType, floor int, floorLabel string) string {
	tier := getUnifiedFloorTier(floor)
	// Unified tower: Mirror Halls (1-10), Crucible of Races (11-25), Labyrinth of Lies (26-50),
	// Gauntlet of Gods (51-75), The Threshold (76-99), The Summit (100)
	var zoneName string
	switch tier {
	case 1:
		zoneName = "Mirror Halls"
	case 2:
		zoneName = "Crucible of Races"
	case 3:
		zoneName = "Labyrinth of Lies"
	case 4:
		zoneName = "Gauntlet of Gods"
	case 5:
		zoneName = "The Threshold"
	default:
		zoneName = "The Summit"
	}

	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("%s Passage (%s)", zoneName, floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("%s Chamber (%s)", zoneName, floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("%s Alcove (%s)", zoneName, floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("%s Ascent (%s)", zoneName, floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("%s Descent (%s)", zoneName, floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("%s Treasury (%s)", zoneName, floorLabel)
	case wfc.TileBoss:
		if tier == 6 {
			return fmt.Sprintf("The Summit (%s)", floorLabel)
		}
		return fmt.Sprintf("%s Guardian Chamber (%s)", zoneName, floorLabel)
	default:
		return fmt.Sprintf("%s (%s)", zoneName, floorLabel)
	}
}

func generateDefaultRoomName(tt wfc.TileType, floorLabel string) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Tower Corridor (%s)", floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("Tower Chamber (%s)", floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Dead End (%s)", floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Ascending Stairway (%s)", floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Descending Stairway (%s)", floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("Treasure Room (%s)", floorLabel)
	case wfc.TileBoss:
		return fmt.Sprintf("Boss Chamber (%s)", floorLabel)
	default:
		return fmt.Sprintf("Unknown Room (%s)", floorLabel)
	}
}

// generateRoomDescription creates a description for a room based on tower theme and floor
func generateRoomDescription(tower string, tt wfc.TileType, floor int) string {
	switch tower {
	case "human":
		return generateHumanDescription(tt, floor)
	case "elf":
		return generateElfDescription(tt, floor)
	case "dwarf":
		return generateDwarfDescription(tt, floor)
	case "gnome":
		return generateGnomeDescription(tt, floor)
	case "orc":
		return generateOrcDescription(tt, floor)
	case "unified":
		return generateUnifiedDescription(tt, floor)
	default:
		return generateDefaultDescription(tt)
	}
}

func generateHumanDescription(tt wfc.TileType, floor int) string {
	tier := getFloorTier(floor)
	// Human: Shattered Atrium (1-5), Burning Library (6-10), Impossible Gallery (11-15),
	// Whispering Archives (16-20), Void Chambers (21-24), The Archive (25)

	switch tier {
	case 1: // Shattered Atrium - broken marble, animated armor, mad scribblings
		switch tt {
		case wfc.TileCorridor:
			return "A corridor of broken marble, shifting shadows moving across the walls. Mad scribblings cover every surface, the words of researchers who lost their minds."
		case wfc.TileRoom:
			return "A once-grand hall now in ruins. Animated suits of armor stand motionless, waiting to attack. Shattered marble columns litter the floor."
		case wfc.TileDeadEnd:
			return "A dead end filled with debris and strange graffiti. The scribblings seem to move when you're not looking directly at them."
		case wfc.TileTreasure:
			return "A treasure alcove amid the ruins. Magical artifacts glint among the broken marble, protected by dormant enchantments."
		case wfc.TileBoss:
			return "A grand atrium where animated armor has gathered in force. Their empty helms turn toward you as one."
		}
	case 2: // Burning Library - eternal fire, fire elementals
		switch tt {
		case wfc.TileCorridor:
			return "A corridor of burning books. Fire consumes the shelves but nothing burns away. The heat grows more intense as you proceed."
		case wfc.TileRoom:
			return "A library chamber engulfed in eternal flames. Books burn endlessly, their knowledge feeding the fire. The flames seem to grow stronger near you."
		case wfc.TileDeadEnd:
			return "A dead end where flames have pooled. The fire here burns hotter, as if concentrated by some malevolent force."
		case wfc.TileTreasure:
			return "A fireproof vault containing books that refused to burn. Their covers are warm to the touch but the knowledge within is intact."
		case wfc.TileBoss:
			return "The heart of the Burning Library, where fire elementals dance among scorched librarians. The heat is nearly unbearable."
		}
	case 3: // Impossible Gallery - defies geometry, gravity shifts
		switch tt {
		case wfc.TileCorridor:
			return "A corridor that defies geometry. The walls curve in impossible ways, and your sense of direction fails completely."
		case wfc.TileRoom:
			return "A gallery where stairs lead to ceilings that become floors. Portraits of former archmages watch you with eyes that move."
		case wfc.TileDeadEnd:
			return "A dead end where space folds in on itself. Looking back, the corridor you came from has changed completely."
		case wfc.TileTreasure:
			return "A treasure room existing in multiple places at once. Reaching for items here requires accepting that distance is meaningless."
		case wfc.TileBoss:
			return "The Gallery's nexus, where gravity pulls in every direction at once. Portraits reach from their frames with grasping hands."
		}
	case 4: // Whispering Archives - floating tomes, weaponized knowledge
		switch tt {
		case wfc.TileCorridor:
			return "A passage lined with floating books that read themselves aloud. The languages cause pain to hear, and understanding brings madness."
		case wfc.TileRoom:
			return "An archive where forbidden tomes drift through the air. Knowledge here is weaponized—learning the wrong thing can reshape your mind."
		case wfc.TileDeadEnd:
			return "A corner where the whispers are loudest. Books have gathered here, their murmuring chorus almost hypnotic."
		case wfc.TileTreasure:
			return "A sealed vault of the most dangerous knowledge. The books here are chained for good reason."
		case wfc.TileBoss:
			return "The central archive, where the whispers form a cacophony. Tomes of pure corruption orbit a nexus of forbidden knowledge."
		}
	case 5: // Void Chambers - reality breakdown, idea creatures
		switch tt {
		case wfc.TileCorridor:
			return "A passage through broken reality. The walls flicker between existence and void, and creatures of pure concept prowl the darkness."
		case wfc.TileRoom:
			return "A chamber in constant flux, its layout changing based on your thoughts. Ideas given form hunt here—concepts that have learned to kill."
		case wfc.TileDeadEnd:
			return "A pocket of void where nothing should exist. Yet something does—something that was once merely a thought."
		case wfc.TileTreasure:
			return "A cache of impossible artifacts—items that defy the laws of reality, born from the void itself."
		case wfc.TileBoss:
			return "The deepest void, where reality has surrendered entirely. What waits here is not creature but concept made manifest."
		}
	default: // The Archive (Floor 25) - Archivist's domain
		switch tt {
		case wfc.TileBoss:
			return "The Archive—the domain of the Archivist. Pure information swirls in visible currents. The being that was once Head Librarian Seraphina waits here, having become a walking paradox of forbidden knowledge."
		default:
			return "The threshold of the Archive, where knowledge itself becomes tangible. The Archivist's presence permeates everything."
		}
	}

	// Default fallback
	switch tt {
	case wfc.TileStairsUp:
		return "A crystalline staircase ascends, glowing runes lighting each step. The magical energy intensifies above."
	case wfc.TileStairsDown:
		return "A spiraling descent through arcane architecture. A shimmering portal offers quick travel to floors you've visited."
	default:
		return "You are in the Arcane Spire, where knowledge became madness."
	}
}

func generateElfDescription(tt wfc.TileType, floor int) string {
	tier := getFloorTier(floor)
	// Elf: Rotting Roots (1-5), Hollow Trunk (6-10), Canker Heart (11-15),
	// Twisted Branches (16-20), Crown of Thorns (21-24), Heart Chamber (25)

	switch tier {
	case 1: // Rotting Roots - corrupted root system, tainted sap pools, twisted animals
		switch tt {
		case wfc.TileCorridor:
			return "A passage through corrupted roots, tainted sap pooling in crevices. Creatures that were once forest animals skulk in the shadows, twisted into something predatory."
		case wfc.TileRoom:
			return "A cavern formed by rotting roots. Pools of corruption block some passages, and the twisted remains of wildlife watch from the darkness."
		case wfc.TileDeadEnd:
			return "A dead end where corruption has pooled deep. The roots here pulse with sickly bioluminescence."
		case wfc.TileTreasure:
			return "An ancient elven cache hidden among the roots. The treasures are tarnished but intact, protected by failing wards."
		case wfc.TileBoss:
			return "The deepest root chamber, where the corruption first took hold. Twisted beasts have made their lair here."
		}
	case 2: // Hollow Trunk - carved chambers, acid sap, blight-touched animals
		switch tt {
		case wfc.TileCorridor:
			return "A tunnel through the hollow trunk, walls weeping corrupted sap that burns like acid. The air itself is toxic."
		case wfc.TileRoom:
			return "An elven chamber carved into living wood, now blackened and diseased. Blight-touched creatures defend their territory with mindless ferocity."
		case wfc.TileDeadEnd:
			return "A sealed chamber where the bark has grown over the entrance. Corruption seeps through the cracks."
		case wfc.TileTreasure:
			return "A forgotten shrine where elves once prayed. The offerings have been corrupted, but power remains."
		case wfc.TileBoss:
			return "The trunk's heart, where disease pumps through the tree like blood. Plant horrors and corrupted beasts guard the passage upward."
		}
	case 3: // Canker Heart - new corrupted life forms, thinking fungi, hunting vines
		switch tt {
		case wfc.TileCorridor:
			return "A passage through living corruption—fungi that think, vines that hunt. The infection has created entirely new forms of life here."
		case wfc.TileRoom:
			return "A chamber of horrors where corruption has spawned abominations. Things that might once have been elves move in the shadows, transformed beyond recognition."
		case wfc.TileDeadEnd:
			return "A pocket where corruption breeds. Spores drift in the air, and tendrils reach from the walls."
		case wfc.TileTreasure:
			return "A node of concentrated corruption containing items of terrible power. The infection seems to be protecting them."
		case wfc.TileBoss:
			return "The canker's core, where the disease thinks and plans. The corruption here is almost sentient."
		}
	case 4: // Twisted Branches - impossible geometries, multi-location chambers
		switch tt {
		case wfc.TileCorridor:
			return "A branch passage that spirals inward impossibly. The disease has reshaped the tree's geometry into something that defies nature."
		case wfc.TileRoom:
			return "A chamber existing in multiple places at once. The creatures here have adapted to this strange space, flickering between locations."
		case wfc.TileDeadEnd:
			return "A fold in corrupted space where the branch loops back on itself. The air shimmers with wrongness."
		case wfc.TileTreasure:
			return "Treasures caught between realities, visible but hard to grasp. The corruption has made distance meaningless here."
		case wfc.TileBoss:
			return "The nexus of twisted branches, where space itself is diseased. Creatures phase in and out of existence as they attack."
		}
	case 5: // Crown of Thorns - thorns and poison, barbs with blight
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the corrupted crown, every surface covered in poisoned thorns. A single scratch delivers the blight."
		case wfc.TileRoom:
			return "A thorn chamber where the corruption is strongest. The barbs seem to reach for you, delivering doses of disease with every touch."
		case wfc.TileDeadEnd:
			return "A corner of concentrated thorns, the poison here visible as a sickly mist."
		case wfc.TileTreasure:
			return "A cache of elven treasures embedded in thorns. Claiming them means accepting the blight's touch."
		case wfc.TileBoss:
			return "The crown's heart, where thorns form a throne of poison. The guardians here are more thorn than flesh."
		}
	default: // Heart Chamber (Floor 25) - Blighted One's domain
		switch tt {
		case wfc.TileBoss:
			return "The Heart Chamber—where the World Tree's life force was once strongest. The Blighted One waits here, corruption given consciousness, embodying the decay that consumes the tree."
		default:
			return "The threshold of the Heart Chamber. The walls pulse with a sickly rhythm, almost like a heartbeat."
		}
	}

	// Default fallback
	switch tt {
	case wfc.TileStairsUp:
		return "A spiral carved into diseased wood ascends. Dark veins pulse in the bark around you."
	case wfc.TileStairsDown:
		return "A winding descent through corrupted wood. A shimmering portal offers quick travel to floors you've visited."
	default:
		return "You are within the Diseased World Tree."
	}
}

func generateDwarfDescription(tt wfc.TileType, floor int) string {
	tier := getFloorTier(floor)
	// Dwarf: Sealed Shafts (1-5), Flooded Galleries (6-10), Mithril Veins (11-15),
	// The Collapse (16-20), The Breach (21-24), The Deep (25)

	switch tier {
	case 1: // Sealed Shafts - warded, watching faces in stone, animated mining equipment
		switch tt {
		case wfc.TileCorridor:
			return "A mine shaft heavily warded with protective runes. The stone walls show faces that weren't carved—they formed on their own, watching."
		case wfc.TileRoom:
			return "A sealed excavation chamber where mining equipment has animated itself. Picks and drills move with hostile purpose."
		case wfc.TileDeadEnd:
			return "A warded dead end where the dwarves tried to seal something away. The runes still glow, but they're weakening."
		case wfc.TileTreasure:
			return "A secure ore cache behind protective wards. The treasures here were deemed too dangerous to move further up."
		case wfc.TileBoss:
			return "The first seal chamber, where animated constructs guard against intrusion from below."
		}
	case 2: // Flooded Galleries - tainted water, eyeless water creatures
		switch tt {
		case wfc.TileCorridor:
			return "A flooded passage where tainted water reaches your knees. Things swim in the darkness—eyeless creatures that need no light to find prey."
		case wfc.TileRoom:
			return "A drowned gallery where underground rivers were diverted. The water corrupts anything it touches over time. Pale shapes move beneath the surface."
		case wfc.TileDeadEnd:
			return "A flooded dead end where the water is deepest. Something large stirs in the depths."
		case wfc.TileTreasure:
			return "A waterlogged vault where treasures lie submerged. The corruption hasn't reached the sealed chests—yet."
		case wfc.TileBoss:
			return "The central reservoir, where aquatic horrors have made their nest. The water here glows with corruption."
		}
	case 3: // Mithril Veins - corrupted mithril, independent shadows
		switch tt {
		case wfc.TileCorridor:
			return "A passage through mithril veins—once the most valuable section of the mines. The metal has been corrupted, burning any who touch it."
		case wfc.TileRoom:
			return "A mithril excavation chamber where shadows move independently of light. The corrupted ore pulses with malevolent energy."
		case wfc.TileDeadEnd:
			return "A pocket of concentrated corrupted mithril. The shadows here are thick and hungry."
		case wfc.TileTreasure:
			return "A mithril cache where some ore remains pure. It gleams defiantly against the surrounding corruption."
		case wfc.TileBoss:
			return "The richest vein, now the most corrupted. Shadows coalesce into solid form here, defending their territory."
		}
	case 4: // The Collapse - unstable stone, shifting tunnels, chaos-adapted creatures
		switch tt {
		case wfc.TileCorridor:
			return "A passage through unstable stone. The tunnels shift and change, and gravity pulls in unpredictable directions."
		case wfc.TileRoom:
			return "A chamber where the corruption has destabilized reality itself. The walls move, the floor tilts, and the creatures here have adapted to constant chaos."
		case wfc.TileDeadEnd:
			return "A collapsed section where stone moves like liquid. There is no safe footing here."
		case wfc.TileTreasure:
			return "Treasures caught in the collapse, visible through shifting stone. Claiming them requires timing and luck."
		case wfc.TileBoss:
			return "The heart of the collapse, where reality churns like a maelstrom. The creatures here exist in multiple configurations at once."
		}
	case 5: // The Breach - original contact point, thick corruption, light-devouring darkness
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the breach zone, where dwarven picks first struck something other. The darkness here is absolute—it devours light."
		case wfc.TileRoom:
			return "A chamber at the edge of the breach. The corruption is thick in the air, seeping from the walls. Your torch barely penetrates the hungry darkness."
		case wfc.TileDeadEnd:
			return "A pocket of primordial darkness. Your light dies completely here, leaving only the sensation of something watching."
		case wfc.TileTreasure:
			return "Treasures left by those who tried to seal the breach. They failed, but their offerings remain, corrupted but powerful."
		case wfc.TileBoss:
			return "The threshold of the breach, where the last defenders fell. The darkness here moves with purpose."
		}
	default: // The Deep (Floor 25) - Deep Guardian's domain
		switch tt {
		case wfc.TileBoss:
			return "The Deep—where the Deep Guardian waits. This massive corrupted construct, three stories tall, was built to seal the breach but was transformed by the darkness it was meant to contain. Its adamantine plates and inverted runes burn with sickly fire."
		default:
			return "The threshold of the Deep. The darkness here is not absence of light—it is a presence, ancient and hungry."
		}
	}

	// Default fallback
	switch tt {
	case wfc.TileStairsUp:
		return "A carved shaft leading back toward the surface. The air grows slightly fresher, and the darkness recedes."
	case wfc.TileStairsDown:
		return "A deep shaft descending into greater darkness. A shimmering portal offers quick travel to levels you've explored."
	default:
		return "You are in the Descending Mines, where dwarves dug too deep."
	}
}

func generateGnomeDescription(tt wfc.TileType, floor int) string {
	tier := getFloorTier(floor)
	// Gnome: Assembly Lines (1-5), Steam Works (6-10), Calculation Engines (11-15),
	// Prototype Labs (16-20), Master Forge (21-24), The Core (25)

	switch tier {
	case 1: // Assembly Lines - conveyor belts, mechanical arms, corrupted automatons
		switch tt {
		case wfc.TileCorridor:
			return "A production corridor where conveyor belts carry components past mechanical arms. The machines now assemble weapons instead of helpful devices."
		case wfc.TileRoom:
			return "An assembly floor where corrupted automatons are born. Mechanical arms work with terrifying precision, and the products are designed for one purpose: violence."
		case wfc.TileDeadEnd:
			return "A maintenance bay where half-assembled automatons wait. Some twitch with partial activation, reaching for intruders."
		case wfc.TileTreasure:
			return "A component storage room containing rare parts. The tower hasn't noticed these supplies yet—they could be salvaged."
		case wfc.TileBoss:
			return "The main assembly hub, where the production line's output has gathered. Corrupted automatons defend their birthplace."
		}
	case 2: // Steam Works - boilers, turbines, scalding steam, weaponized equipment
		switch tt {
		case wfc.TileCorridor:
			return "A steam-filled passage between massive boilers. The heat is almost unbearable, and jets of superheated steam target intruders."
		case wfc.TileRoom:
			return "A turbine chamber where the tower's power is generated. The machines have been modified to actively attack, turning industrial equipment into weapons."
		case wfc.TileDeadEnd:
			return "A pressure relief alcove where steam has pooled dangerously. The vents here seem to track movement."
		case wfc.TileTreasure:
			return "An engineer's vault containing heat-resistant materials and tools. The contents survived the tower's transformation."
		case wfc.TileBoss:
			return "The main boiler room, where steam pressure reaches critical levels. The tower's industrial heart defends itself with scalding fury."
		}
	case 3: // Calculation Engines - computational machinery, coordinated attacks
		switch tt {
		case wfc.TileCorridor:
			return "A passage through banks of computational machinery. Clicking and whirring fills the air as the tower thinks, plans, calculates."
		case wfc.TileRoom:
			return "A calculation chamber forming part of the tower's distributed brain. The automatons here are smarter, coordinating attacks with unsettling precision."
		case wfc.TileDeadEnd:
			return "A processing node where equations scroll across every surface. The tower is particularly aware of intruders here."
		case wfc.TileTreasure:
			return "A data archive containing the tower's original blueprints. Understanding these could reveal vulnerabilities."
		case wfc.TileBoss:
			return "A central processing hub where the tower's intelligence concentrates. The defenses here anticipate your every move."
		}
	case 4: // Prototype Labs - unstable machines, experimental failures
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the prototype section. Unstable machines line the walls—failed experiments that were never meant to leave testing."
		case wfc.TileRoom:
			return "A testing chamber filled with experimental constructs. These prototypes are unpredictable, fighting with the desperation of things that know they shouldn't exist."
		case wfc.TileDeadEnd:
			return "A containment alcove where a particularly dangerous prototype was isolated. Its cage shows signs of recent damage."
		case wfc.TileTreasure:
			return "A prototype vault containing experimental designs. Some failures hold valuable innovations."
		case wfc.TileBoss:
			return "The main testing arena, where the tower's most ambitious failures have gathered. Unpredictable and deadly."
		}
	case 5: // Master Forge - liquid metal, organic-looking machines
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the Master Forge, where raw materials become automatons. Metal flows like liquid here, forming shapes that mimic life."
		case wfc.TileRoom:
			return "A forge chamber where the creation process has become almost organic. The automatons born here are nearly indistinguishable from living beings."
		case wfc.TileDeadEnd:
			return "A cooling alcove where newly-forged automatons take final shape. The metal here still ripples like flesh."
		case wfc.TileTreasure:
			return "A materials vault containing the rarest metals. The tower's masterpiece creations were forged from these."
		case wfc.TileBoss:
			return "The heart of the Master Forge, where the tower creates its most sophisticated servants. The automatons here are works of terrible art."
		}
	default: // The Core (Floor 25) - Prime Calculation's domain
		switch tt {
		case wfc.TileBoss:
			return "The Core—a perfect sphere of polished metal where the Prime Calculation waits. Equations shift across every surface. This is not a creature but an intelligence, a mathematical entity that emerged from the tower's systems. It cannot be killed—only out-thought."
		default:
			return "The threshold of the Core. The walls here are mirrors of polished metal, and your reflection seems to calculate your weaknesses."
		}
	}

	// Default fallback
	switch tt {
	case wfc.TileStairsUp:
		return "A mechanical lift ascends through grinding gears. The tower's machinery watches your progress."
	case wfc.TileStairsDown:
		return "An elevator platform descends from above. A shimmering portal offers quick travel to floors you've visited."
	default:
		return "You are in the Mechanical Tower, where progress devoured itself."
	}
}

func generateOrcDescription(tt wfc.TileType, floor int) string {
	tier := getFloorTier(floor)
	// Orc: Ossuary (1-5), Champions' Rest (6-10), Proving Grounds (11-15),
	// Hall of Chieftains (16-20), Beast's Spine (21-24), Skull Throne (25)

	switch tier {
	case 1: // Ossuary - common warrior tombs, animated skeletons, watching skulls
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the Ossuary, walls lined with skulls that watch and whisper. The bones of common warriors rattle in their alcoves."
		case wfc.TileRoom:
			return "A tomb chamber where common warriors were laid to rest. Their bones now animate, forming skeletal warriors that remember enough of their skills to be deadly."
		case wfc.TileDeadEnd:
			return "A bone alcove where skulls have piled deep. They watch you with empty sockets, and some begin to move."
		case wfc.TileTreasure:
			return "A warrior's grave cache, offerings to the honored dead. The spirits seem reluctant to let you claim them."
		case wfc.TileBoss:
			return "The Ossuary's heart, where the first awakened dead have gathered. Skeletal warriors form ranks, ready for battle."
		}
	case 2: // Champions' Rest - renowned warriors, intelligent dead, animated trophies
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the Champions' Rest, where warriors of renown were interred. These dead are more dangerous—they coordinate their attacks with intelligence."
		case wfc.TileRoom:
			return "A champion's tomb filled with trophies of victory. The dead here retain not just skill but tactics. The trophies on the walls sometimes join the fight."
		case wfc.TileDeadEnd:
			return "A trophy alcove where a great warrior's prizes are displayed. The armor and weapons here seem eager to find new wielders."
		case wfc.TileTreasure:
			return "A champion's hoard, the accumulated wealth of a legendary warrior. The guardian's spirit watches jealously."
		case wfc.TileBoss:
			return "The greatest champion's tomb, where the most renowned dead hold court. They fight as they did in life—with terrifying skill."
		}
	case 3: // Proving Grounds - manifested illusions, creatures of revenge
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the Proving Grounds, where warriors once tested themselves against illusions. The illusions have become real now."
		case wfc.TileRoom:
			return "An arena where ancient foes have manifested. The spirits have turned this proving ground into a gauntlet of revenge—every creature the orcs ever defeated fights again."
		case wfc.TileDeadEnd:
			return "A meditation alcove where warriors prepared for trials. The spirits of their old enemies wait here now."
		case wfc.TileTreasure:
			return "A victor's vault, rewards for those who passed the trials. The treasures are guarded by the memory of past challenges."
		case wfc.TileBoss:
			return "The final trial arena, where all the proving grounds' horrors converge. The spirits demand you prove your worth."
		}
	case 4: // Hall of Chieftains - legendary leaders, speaking spirits, demanding answers
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the Hall of Chieftains, where legendary war leaders were laid to rest. These spirits are fully aware, and they demand answers."
		case wfc.TileRoom:
			return "A chieftain's throne room, where the dead leader still commands. The spirit can speak and reason—though reason has not made them merciful."
		case wfc.TileDeadEnd:
			return "A war council chamber where chieftains planned their campaigns. Their spirits still debate strategy, and they're eager for fresh tactics."
		case wfc.TileTreasure:
			return "A chieftain's treasury, the wealth of a legendary leader. The spirit watches to see if you're worthy to claim it."
		case wfc.TileBoss:
			return "The great chieftain's hall, where the mightiest war leader holds eternal court. This spirit commanded armies in life and commands the dead in death."
		}
	case 5: // Beast's Spine - actual vertebrae, awakened beast spirit, attacking bone
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the Beast's Spine—actual vertebrae of the great creature whose skull crowns the tower. The bone itself attacks intruders."
		case wfc.TileRoom:
			return "A chamber within the beast's skeleton, where its spirit has partially awakened. Not as a thinking being, but as pure animal rage."
		case wfc.TileDeadEnd:
			return "A bone pocket where the beast's essence has pooled. Spikes form from the walls, and the passage tries to crush you."
		case wfc.TileTreasure:
			return "A cache lodged within the beast's bones. Ancient offerings to the creature, preserved within its skeleton."
		case wfc.TileBoss:
			return "The beast's heart chamber, where the great creature's rage is strongest. The bones move with terrible purpose."
		}
	default: // Skull Throne (Floor 25) - Ancestor King's domain
		switch tt {
		case wfc.TileBoss:
			return "The Skull Throne—within the great skull itself, where the Ancestor King holds court. The first orc, whose spirit has been bound here so long he has forgotten what it was to be alive. He demands answers for a broken promise, a betrayal the living cannot remember."
		default:
			return "The threshold of the Skull Throne. The beast's skull looms above, and the presence of the Ancestor King weighs upon you."
		}
	}

	// Default fallback
	switch tt {
	case wfc.TileStairsUp:
		return "A warrior's ascent marked with victory runes. Only the strong climb higher in the Beast-Skull Tower."
	case wfc.TileStairsDown:
		return "Blood-stained stairs descend from above. A shimmering portal offers quick travel to floors you've conquered."
	default:
		return "You are in the Beast-Skull Tower, where the dead refuse to rest."
	}
}

func generateUnifiedDescription(tt wfc.TileType, floor int) string {
	tier := getUnifiedFloorTier(floor)
	// Unified: Mirror Halls (1-10), Crucible of Races (11-25), Labyrinth of Lies (26-50),
	// Gauntlet of Gods (51-75), The Threshold (76-99), The Summit (100)

	switch tier {
	case 1: // Mirror Halls - facing versions of yourself, past and possible selves
		switch tt {
		case wfc.TileCorridor:
			return "A corridor of mirrors reflecting versions of yourself—past selves, possible selves, selves that made different choices. They watch with familiar eyes."
		case wfc.TileRoom:
			return "A hall of reflection where mirror-selves wait. They are as strong as you and know every technique you know. Victory requires growing beyond who you were."
		case wfc.TileDeadEnd:
			return "A corner of mirrors where your reflections have gathered. They whisper of choices you didn't make, paths you didn't take."
		case wfc.TileTreasure:
			return "A mirror chamber holding treasures from other possibilities—items you might have earned, had you made different choices."
		case wfc.TileBoss:
			return "The central mirror chamber, where your greatest self waits. Not who you are, but who you could have been."
		}
	case 2: // Crucible of Races - combined corruption from all five towers
		switch tt {
		case wfc.TileCorridor:
			return "A passage where the corruption of all five towers has merged. Fire and decay, mechanism and bone, darkness and madness—all concentrated here."
		case wfc.TileRoom:
			return "A crucible chamber where challenges from every racial tower have been remixed and intensified. The threats here are familiar yet more deadly."
		case wfc.TileDeadEnd:
			return "A pocket where multiple corruptions have pooled. The energies war with each other, creating unpredictable dangers."
		case wfc.TileTreasure:
			return "A vault containing artifacts from all five towers. The combined corruption has created items of terrible power."
		case wfc.TileBoss:
			return "The crucible's heart, where champions from every race have fallen. Their combined strength guards the way forward."
		}
	case 3: // Labyrinth of Lies - illusions indistinguishable from truth, unreliable reality
		switch tt {
		case wfc.TileCorridor:
			return "A passage where reality is unreliable. Illusions are indistinguishable from truth. What you see may not exist, and what exists may be hidden."
		case wfc.TileRoom:
			return "A labyrinth chamber where nothing is certain. Allies may be enemies in disguise; enemies may be potential allies. Trust is the Spire's weapon here."
		case wfc.TileDeadEnd:
			return "A dead end—or is it? The walls shimmer with illusion. Truth and lie are indistinguishable in this place."
		case wfc.TileTreasure:
			return "Treasures that may be real or may be bait. Many champions have lost themselves reaching for prizes that never existed."
		case wfc.TileBoss:
			return "The labyrinth's heart, where the greatest deception waits. Nothing here is as it appears—including the way forward."
		}
	case 4: // Gauntlet of Gods - divine avatars, impossible knowledge, soul sacrifices
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the divine gauntlet. The tests here push the limits of mortal capability. Avatars of divine power walk these halls."
		case wfc.TileRoom:
			return "A trial chamber where the gods themselves set the challenges. Puzzles requiring knowledge no mortal should possess. Tests demanding pieces of your very soul."
		case wfc.TileDeadEnd:
			return "A shrine to fallen challengers. Those who could not meet the divine standard are remembered here—what remains of them."
		case wfc.TileTreasure:
			return "An offering from the gods themselves—power earned through sacrifice. The cost is written on the faces of those who paid it."
		case wfc.TileBoss:
			return "An arena where divine avatars wait in single combat. These are not the gods themselves, but they are close enough."
		}
	case 5: // The Threshold - rules break down, personalized tests, beings who know the end
		switch tt {
		case wfc.TileCorridor:
			return "A passage through the Threshold, where the rules break down entirely. This corridor is different for each who walks it, designed to test what remains untested."
		case wfc.TileRoom:
			return "A chamber shaped by your deepest fears and highest hopes. The Spire knows you intimately here, and it uses that knowledge without mercy."
		case wfc.TileDeadEnd:
			return "A corner where you face impossible choices. There is no right answer—only revelations about who you truly are."
		case wfc.TileTreasure:
			return "Rewards tailored to your specific desires. The Spire offers everything you've ever wanted. The price is everything you are."
		case wfc.TileBoss:
			return "A chamber where beings speak who claim to know how everything ends. Their words cut deeper than any blade."
		}
	default: // The Summit (Floor 100) - The Architect's domain
		switch tt {
		case wfc.TileBoss:
			return "The Summit—where no one has ever reached. A door waits here, a door that would change everything if opened. The Architect watches through the Spire itself, designing one final test: the choice of whether to open it."
		default:
			return "The threshold of the Summit. You stand at the edge of everything. The air is charged with possibility—and danger."
		}
	}

	// Default fallback
	switch tt {
	case wfc.TileStairsUp:
		return "An ascent through crystallized possibility. The Spire shifts around you, adapting to your presence."
	case wfc.TileStairsDown:
		return "A descent through the Spire's depths. A shimmering portal offers escape to floors you've survived."
	default:
		return "You are within the Infinity Spire, where everything is tested and nothing is certain."
	}
}

func generateDefaultDescription(tt wfc.TileType) string {
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
