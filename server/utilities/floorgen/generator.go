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
			Description: generateRoomDescription(g.TowerID, tile.Type),
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

// generateRoomName creates a name for a room based on tower theme and tile type
func generateRoomName(tower string, tt wfc.TileType, floor int) string {
	// Dwarf tower uses "Level" instead of "Floor" and descending terminology
	floorLabel := fmt.Sprintf("Floor %d", floor)
	if tower == "dwarf" {
		floorLabel = fmt.Sprintf("Mine Level %d", floor)
	} else if tower == "unified" {
		floorLabel = fmt.Sprintf("Spire %d", floor)
	}

	switch tower {
	case "human":
		return generateHumanRoomName(tt, floorLabel)
	case "elf":
		return generateElfRoomName(tt, floorLabel)
	case "dwarf":
		return generateDwarfRoomName(tt, floorLabel)
	case "gnome":
		return generateGnomeRoomName(tt, floorLabel)
	case "orc":
		return generateOrcRoomName(tt, floorLabel)
	case "unified":
		return generateUnifiedRoomName(tt, floorLabel)
	default:
		return generateDefaultRoomName(tt, floorLabel)
	}
}

func generateHumanRoomName(tt wfc.TileType, floorLabel string) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Arcane Passage (%s)", floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("Enchanted Chamber (%s)", floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Spell-Sealed Alcove (%s)", floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Ascending Stairway (%s)", floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Descending Stairway (%s)", floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("Arcane Treasury (%s)", floorLabel)
	case wfc.TileBoss:
		return fmt.Sprintf("Grand Arcanum (%s)", floorLabel)
	default:
		return fmt.Sprintf("Unknown Room (%s)", floorLabel)
	}
}

func generateElfRoomName(tt wfc.TileType, floorLabel string) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Blighted Tunnel (%s)", floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("Infected Hollow (%s)", floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Diseased Alcove (%s)", floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Corrupted Ascent (%s)", floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Spiral Descent (%s)", floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("Forgotten Shrine (%s)", floorLabel)
	case wfc.TileBoss:
		return fmt.Sprintf("Heart of the Blight (%s)", floorLabel)
	default:
		return fmt.Sprintf("Unknown Room (%s)", floorLabel)
	}
}

func generateDwarfRoomName(tt wfc.TileType, floorLabel string) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Mine Shaft (%s)", floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("Excavated Cavern (%s)", floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Collapsed Tunnel (%s)", floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Shaft to Surface (%s)", floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Shaft to Depths (%s)", floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("Ore Vein Chamber (%s)", floorLabel)
	case wfc.TileBoss:
		return fmt.Sprintf("Deep Guardian's Lair (%s)", floorLabel)
	default:
		return fmt.Sprintf("Unknown Room (%s)", floorLabel)
	}
}

func generateGnomeRoomName(tt wfc.TileType, floorLabel string) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Maintenance Corridor (%s)", floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("Machine Chamber (%s)", floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Service Alcove (%s)", floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Elevator Shaft Up (%s)", floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Elevator Shaft Down (%s)", floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("Component Storage (%s)", floorLabel)
	case wfc.TileBoss:
		return fmt.Sprintf("Central Processing (%s)", floorLabel)
	default:
		return fmt.Sprintf("Unknown Room (%s)", floorLabel)
	}
}

func generateOrcRoomName(tt wfc.TileType, floorLabel string) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Bone Passage (%s)", floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("Trophy Hall (%s)", floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Skull Alcove (%s)", floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Warrior's Ascent (%s)", floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Blood Stair (%s)", floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("War Trophy Chamber (%s)", floorLabel)
	case wfc.TileBoss:
		return fmt.Sprintf("Warchief's Arena (%s)", floorLabel)
	default:
		return fmt.Sprintf("Unknown Room (%s)", floorLabel)
	}
}

func generateUnifiedRoomName(tt wfc.TileType, floorLabel string) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Corrupted Passage (%s)", floorLabel)
	case wfc.TileRoom:
		return fmt.Sprintf("Blighted Chamber (%s)", floorLabel)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Festering Alcove (%s)", floorLabel)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Spiral of Decay (%s)", floorLabel)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Descent of Corruption (%s)", floorLabel)
	case wfc.TileTreasure:
		return fmt.Sprintf("Tainted Treasury (%s)", floorLabel)
	case wfc.TileBoss:
		return fmt.Sprintf("Throne of Corruption (%s)", floorLabel)
	default:
		return fmt.Sprintf("Unknown Room (%s)", floorLabel)
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

// generateRoomDescription creates a description for a room based on tower theme
func generateRoomDescription(tower string, tt wfc.TileType) string {
	switch tower {
	case "human":
		return generateHumanDescription(tt)
	case "elf":
		return generateElfDescription(tt)
	case "dwarf":
		return generateDwarfDescription(tt)
	case "gnome":
		return generateGnomeDescription(tt)
	case "orc":
		return generateOrcDescription(tt)
	case "unified":
		return generateUnifiedDescription(tt)
	default:
		return generateDefaultDescription(tt)
	}
}

func generateHumanDescription(tt wfc.TileType) string {
	switch tt {
	case wfc.TileCorridor:
		return "An arcane passage lined with glowing runes. The air crackles with residual magic, and strange symbols pulse with inner light."
	case wfc.TileRoom:
		return "An enchanted chamber where magical experiments once took place. Crystal formations grow from the walls, humming with stored energy."
	case wfc.TileDeadEnd:
		return "A spell-sealed alcove containing ancient magical inscriptions. The air feels thick with dormant enchantments."
	case wfc.TileStairsUp:
		return "A spiraling staircase of pure crystal ascends into crackling magical energy. The steps glow beneath your feet."
	case wfc.TileStairsDown:
		return "A crystalline staircase descends from above. A shimmering portal offers quick travel to floors you've visited."
	case wfc.TileTreasure:
		return "An arcane treasury filled with magical artifacts and glowing crystals. Wards shimmer around the most valuable items."
	case wfc.TileBoss:
		return "A grand arcanum where raw magical power swirls in visible currents. Something powerful lurks here."
	default:
		return "You are in a room within the Arcane Spire."
	}
}

func generateElfDescription(tt wfc.TileType) string {
	switch tt {
	case wfc.TileCorridor:
		return "A tunnel through diseased wood, the walls weeping corrupted sap. Bioluminescent fungus provides sickly illumination."
	case wfc.TileRoom:
		return "A hollow within the corrupted World Tree. The bark is blackened and twisted, and the air reeks of decay."
	case wfc.TileDeadEnd:
		return "A diseased alcove where the corruption has pooled. Strange growths cover the walls, pulsing with malevolent life."
	case wfc.TileStairsUp:
		return "A spiraling path carved into the living wood ascends higher into the blighted tree. Dark veins pulse in the bark."
	case wfc.TileStairsDown:
		return "A winding descent through corrupted wood. A shimmering portal offers quick travel to floors you've visited."
	case wfc.TileTreasure:
		return "A forgotten shrine now overgrown with corruption. Treasures of the old world lie buried beneath diseased roots."
	case wfc.TileBoss:
		return "The heart of the blight beats here, a chamber pulsing with corruption. The source of the World Tree's disease awaits."
	default:
		return "You are within the Diseased World Tree."
	}
}

func generateDwarfDescription(tt wfc.TileType) string {
	switch tt {
	case wfc.TileCorridor:
		return "A mine shaft carved through solid rock. Rail tracks run along the floor, and the walls glitter with exposed mineral veins."
	case wfc.TileRoom:
		return "An excavated cavern with rough-hewn walls. Mining equipment lies abandoned, and ore carts rust in the corners."
	case wfc.TileDeadEnd:
		return "A collapsed tunnel blocked by fallen stone. Pick marks on the walls show where miners once worked."
	case wfc.TileStairsUp:
		return "A carved shaft leading back toward the surface. The air grows slightly fresher as you ascend."
	case wfc.TileStairsDown:
		return "A deep shaft descending into the darkness below. A shimmering portal offers quick travel to levels you've explored."
	case wfc.TileTreasure:
		return "An ore vein chamber where precious metals gleam in the torchlight. The richest deposits are guarded by something."
	case wfc.TileBoss:
		return "The deepest excavation, where dwarves dug too deep. Ancient and terrible things lurk in this darkness."
	default:
		return "You are in the Descending Mines."
	}
}

func generateGnomeDescription(tt wfc.TileType) string {
	switch tt {
	case wfc.TileCorridor:
		return "A maintenance corridor lined with pipes and conduits. Steam hisses from joints, and gears spin in exposed mechanisms."
	case wfc.TileRoom:
		return "A machine chamber filled with whirring contraptions. Conveyor belts move endlessly, and indicator lights blink in patterns."
	case wfc.TileDeadEnd:
		return "A service alcove packed with spare parts and maintenance tools. Abandoned automatons stand frozen mid-repair."
	case wfc.TileStairsUp:
		return "An elevator shaft with a rickety mechanical lift. Gears grind as the platform ascends through the tower."
	case wfc.TileStairsDown:
		return "An elevator platform descends from above. A shimmering portal offers quick travel to floors you've visited."
	case wfc.TileTreasure:
		return "A component storage room filled with rare parts and valuable materials. Some machines here still function."
	case wfc.TileBoss:
		return "Central processing - the core of the corrupted machine intelligence. Cables snake across every surface, pulsing with power."
	default:
		return "You are in the Mechanical Tower."
	}
}

func generateOrcDescription(tt wfc.TileType) string {
	switch tt {
	case wfc.TileCorridor:
		return "A passage decorated with bones and skulls. War trophies hang from hooks, and the floor is stained with old blood."
	case wfc.TileRoom:
		return "A trophy hall lined with the remains of fallen enemies. Weapons and armor of the conquered adorn the walls."
	case wfc.TileDeadEnd:
		return "A skull alcove containing a shrine of bones. Offerings of blood and meat rot at its base."
	case wfc.TileStairsUp:
		return "A warrior's ascent marked with victory runes. Only the strong climb higher in the Beast-Skull Tower."
	case wfc.TileStairsDown:
		return "Blood-stained stairs descend from above. A shimmering portal offers quick travel to floors you've conquered."
	case wfc.TileTreasure:
		return "A war trophy chamber filled with plunder from countless battles. The most valued prizes are displayed prominently."
	case wfc.TileBoss:
		return "The warchief's arena, where only the mightiest survive. Bones of challengers litter the blood-soaked floor."
	default:
		return "You are in the Beast-Skull Tower."
	}
}

func generateUnifiedDescription(tt wfc.TileType) string {
	switch tt {
	case wfc.TileCorridor:
		return "A passage of flesh and bone, the walls dripping with corruption. The air is thick with the stench of decay, and strange growths pulse with sickly light."
	case wfc.TileRoom:
		return "A chamber of wrongness where reality itself seems sick. The walls breathe slowly, and whispers echo from nowhere. You feel the presence of ancient corruption."
	case wfc.TileDeadEnd:
		return "A festering alcove where corruption has pooled deep. Strange formations grow from every surface, and the air itself seems to infect your lungs."
	case wfc.TileStairsUp:
		return "A spiral of decay ascending through corrupted flesh. Each step squelches beneath your feet, and the walls seem to watch your ascent."
	case wfc.TileStairsDown:
		return "A descent into deeper corruption. A shimmering portal offers escape to floors you've survived. The darkness below hungers for your presence."
	case wfc.TileTreasure:
		return "A treasury tainted by corruption, treasures of fallen worlds gathered here. Even the gold seems diseased, but power radiates from within."
	case wfc.TileBoss:
		return "A throne room of corruption where the air screams in silence. Something ancient and terrible awaits here, at the heart of all decay."
	default:
		return "You are within the Infinity Spire, where corruption reigns eternal."
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
