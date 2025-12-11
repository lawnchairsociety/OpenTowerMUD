package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// LabyrinthYAML represents the labyrinth in YAML format
type LabyrinthYAML struct {
	Width         int                  `yaml:"width"`
	Height        int                  `yaml:"height"`
	GeneratedSeed int64                `yaml:"generated_seed"`
	Gates         []GateYAML           `yaml:"gates"`
	Shortcuts     []ShortcutYAML       `yaml:"shortcuts"`
	Rooms         map[string]*RoomYAML `yaml:"rooms"`
}

// GateYAML represents a city gate in YAML
type GateYAML struct {
	CityID   string `yaml:"city_id"`
	CityName string `yaml:"city_name"`
	RoomID   string `yaml:"room_id"`
}

// ShortcutYAML represents a shortcut pair in YAML
type ShortcutYAML struct {
	RoomA string `yaml:"room_a"`
	RoomB string `yaml:"room_b"`
}

// RoomYAML represents a room in YAML format
type RoomYAML struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"`
	Features    []string          `yaml:"features,omitempty"`
	Exits       map[string]string `yaml:"exits,omitempty"`
}

// WriteYAML converts the maze to YAML and writes it to a file
func (mg *MazeGenerator) WriteYAML(outputDir string) error {
	labyrinth := &LabyrinthYAML{
		Width:         mg.Width,
		Height:        mg.Height,
		GeneratedSeed: mg.Rand.Int63(), // Store the seed
		Gates:         make([]GateYAML, 0, len(mg.Gates)),
		Shortcuts:     make([]ShortcutYAML, 0, len(mg.Shortcuts)),
		Rooms:         make(map[string]*RoomYAML),
	}

	// Add gates
	for _, gate := range mg.Gates {
		labyrinth.Gates = append(labyrinth.Gates, GateYAML{
			CityID:   gate.CityID,
			CityName: gate.CityName,
			RoomID:   getRoomID(gate.X, gate.Y),
		})
	}

	// Add shortcuts
	for _, sc := range mg.Shortcuts {
		labyrinth.Shortcuts = append(labyrinth.Shortcuts, ShortcutYAML{
			RoomA: getRoomID(sc.X1, sc.Y1),
			RoomB: getRoomID(sc.X2, sc.Y2),
		})
	}

	// Convert cells to rooms
	for y := 0; y < mg.Height; y++ {
		for x := 0; x < mg.Width; x++ {
			cell := mg.Grid[y][x]
			roomID := getRoomID(x, y)
			room := mg.cellToRoom(cell)
			labyrinth.Rooms[roomID] = room
		}
	}

	// Write to file
	path := filepath.Join(outputDir, "labyrinth.yaml")
	return writeLabyrinthYAML(labyrinth, path)
}

// getRoomID generates a room ID from coordinates
func getRoomID(x, y int) string {
	return fmt.Sprintf("labyrinth_%d_%d", x, y)
}

// cellToRoom converts a maze cell to a room
func (mg *MazeGenerator) cellToRoom(cell *Cell) *RoomYAML {
	room := &RoomYAML{
		Type:  "labyrinth",
		Exits: make(map[string]string),
	}

	// Set name and description based on cell type/POI
	switch {
	case cell.Type == CellGate:
		gate := mg.getGateForCell(cell)
		room.Name = fmt.Sprintf("%s Gate", gate.CityName)
		room.Description = fmt.Sprintf("A massive archway marks the entrance to the labyrinth from %s. Ancient runes carved into the stone pulse with a faint light. The air here feels different - the boundary between city and labyrinth is almost tangible.", gate.CityName)
		room.Type = "labyrinth_gate"
		room.Features = []string{"gate", "labyrinth_entrance"}

	case cell.POI == POITreasure:
		room.Name = "Hidden Vault"
		room.Description = "A forgotten chamber filled with the remnants of ancient treasures. Dust motes dance in the dim light filtering through cracks in the ceiling. Someone hid their valuables here long ago."
		room.Features = []string{"treasure"}

	case cell.POI == POIMerchant:
		room.Name = "Merchant's Alcove"
		room.Description = "A surprisingly well-maintained alcove where a hooded figure has set up a small trading post. How they get their supplies this deep in the labyrinth is a mystery."
		room.Features = []string{"merchant"}

	case cell.POI == POILoreNPC:
		room.Name = "Scholar's Refuge"
		room.Description = "A quiet corner of the labyrinth where someone has made a small camp. Books and scrolls are scattered about, and strange symbols are drawn on the walls."
		room.Features = []string{"lore_npc"}

	case cell.POI == POIShortcutA || cell.POI == POIShortcutB:
		room.Name = "Mysterious Passage"
		room.Description = "The walls here shimmer with an otherworldly energy. A strange portal hovers in the air, leading to somewhere else in the labyrinth. Ancient magic allows instant travel through these connected points."
		room.Features = []string{"shortcut"}
		// Shortcut exits are added separately

	default:
		room.Name, room.Description = mg.generatePassageNameAndDesc(cell)
	}

	// Build exits from walls
	for _, dir := range AllDirections() {
		if !cell.Walls[dir] {
			nx, ny := mg.neighbor(cell.X, cell.Y, dir)
			if mg.inBounds(nx, ny) {
				room.Exits[dir.String()] = getRoomID(nx, ny)
			}
		}
	}

	// Add shortcut exits
	if cell.POI == POIShortcutA || cell.POI == POIShortcutB {
		for _, sc := range mg.Shortcuts {
			if cell.X == sc.X1 && cell.Y == sc.Y1 {
				room.Exits["portal"] = getRoomID(sc.X2, sc.Y2)
			} else if cell.X == sc.X2 && cell.Y == sc.Y2 {
				room.Exits["portal"] = getRoomID(sc.X1, sc.Y1)
			}
		}
	}

	return room
}

// getGateForCell finds the gate info for a gate cell
func (mg *MazeGenerator) getGateForCell(cell *Cell) *GateInfo {
	for i := range mg.Gates {
		if mg.Gates[i].X == cell.X && mg.Gates[i].Y == cell.Y {
			return &mg.Gates[i]
		}
	}
	return &GateInfo{CityID: "unknown", CityName: "Unknown"}
}

// generatePassageNameAndDesc generates varied names and descriptions for passages
func (mg *MazeGenerator) generatePassageNameAndDesc(cell *Cell) (string, string) {
	// Count exits
	exits := 0
	for _, dir := range AllDirections() {
		if !cell.Walls[dir] {
			exits++
		}
	}

	// Use coordinates to create deterministic variety
	variant := (cell.X + cell.Y*3) % 5

	switch exits {
	case 1:
		// Dead end
		names := []string{"Dead End", "Collapsed Passage", "Blocked Tunnel", "Sealed Alcove", "Rubble-Filled Chamber"}
		descs := []string{
			"The passage ends abruptly here. Ancient stones have fallen to block any further progress.",
			"Rubble fills this end of the tunnel. Whatever lay beyond is now inaccessible.",
			"The walls close in here, with no way forward. Scratches on the stone suggest others have tried to dig through.",
			"A small alcove marks the end of this path. Cobwebs hang thick in the corners.",
			"The tunnel terminates in a pile of collapsed masonry. The air is stale and musty.",
		}
		return names[variant], descs[variant]

	case 2:
		// Corridor
		names := []string{"Winding Passage", "Stone Corridor", "Ancient Tunnel", "Dusty Hallway", "Forgotten Path"}
		descs := []string{
			"A narrow passage winds through the ancient stone. The walls bear the marks of countless travelers.",
			"This corridor stretches into darkness in both directions. The stones are worn smooth by age.",
			"An ancient tunnel carved through solid rock. Strange symbols are barely visible on the walls.",
			"Dust coats every surface of this forgotten hallway. Your footsteps echo eerily.",
			"A path through the labyrinth that few have walked in ages. The silence is oppressive.",
		}
		return names[variant], descs[variant]

	case 3:
		// T-junction
		names := []string{"Junction", "Crossroads", "Three-Way Split", "Branching Passage", "Fork in the Path"}
		descs := []string{
			"The passage branches here, offering multiple routes through the labyrinth.",
			"A crossroads in the ancient maze. Scratched arrows on the walls point in different directions.",
			"Three passages meet at this junction. The air currents hint at the paths ahead.",
			"The tunnel splits here. Each direction looks equally dark and foreboding.",
			"A fork in the winding path. Someone has left old torch stubs at the base of one wall.",
		}
		return names[variant], descs[variant]

	default:
		// Four-way intersection
		names := []string{"Central Chamber", "Grand Intersection", "Four-Way Crossing", "Hub Chamber", "Meeting of Paths"}
		descs := []string{
			"A central chamber where four passages meet. The ceiling rises higher here, giving a sense of space.",
			"A grand intersection in the labyrinth. Worn carvings suggest this was once an important location.",
			"Four passages converge at this crossing. The stone floor is worn into grooves by countless feet.",
			"A hub chamber connecting multiple routes. The air here moves freely, carrying distant sounds.",
			"Four paths meet in this open space. Ancient pillars support the ceiling at each corner.",
		}
		return names[variant], descs[variant]
	}
}

// writeLabyrinthYAML writes the labyrinth to a YAML file with nice formatting
func writeLabyrinthYAML(labyrinth *LabyrinthYAML, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Write header comment
	fmt.Fprintf(f, "# The Great Labyrinth - Connecting all cities\n")
	fmt.Fprintf(f, "# Generated maze: %dx%d grid\n", labyrinth.Width, labyrinth.Height)
	fmt.Fprintf(f, "# Total rooms: %d\n\n", len(labyrinth.Rooms))

	// Create encoder
	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)

	// Build ordered output
	orderedLabyrinth := &orderedLabyrinthYAML{
		Width:         labyrinth.Width,
		Height:        labyrinth.Height,
		GeneratedSeed: labyrinth.GeneratedSeed,
		Gates:         labyrinth.Gates,
		Shortcuts:     labyrinth.Shortcuts,
		Rooms:         sortRooms(labyrinth.Rooms),
	}

	if err := encoder.Encode(orderedLabyrinth); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// orderedLabyrinthYAML is used for serialization with ordered rooms
type orderedLabyrinthYAML struct {
	Width         int            `yaml:"width"`
	Height        int            `yaml:"height"`
	GeneratedSeed int64          `yaml:"generated_seed"`
	Gates         []GateYAML     `yaml:"gates"`
	Shortcuts     []ShortcutYAML `yaml:"shortcuts"`
	Rooms         yaml.Node      `yaml:"rooms"`
}

// sortRooms returns rooms as an ordered YAML node
func sortRooms(rooms map[string]*RoomYAML) yaml.Node {
	// Get sorted room IDs
	ids := make([]string, 0, len(rooms))
	for id := range rooms {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Build ordered map node
	node := yaml.Node{
		Kind: yaml.MappingNode,
	}

	for _, id := range ids {
		room := rooms[id]

		// Room ID key
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: id,
		}

		// Room value as a mapping
		valueNode := yaml.Node{
			Kind: yaml.MappingNode,
		}

		// Add room fields in order
		addStringField(&valueNode, "name", room.Name)
		addStringField(&valueNode, "description", room.Description)
		addStringField(&valueNode, "type", room.Type)

		if len(room.Features) > 0 {
			addSequenceField(&valueNode, "features", room.Features)
		}

		if len(room.Exits) > 0 {
			addMapField(&valueNode, "exits", room.Exits)
		}

		node.Content = append(node.Content, &keyNode, &valueNode)
	}

	return node
}

func addStringField(node *yaml.Node, key, value string) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

func addSequenceField(node *yaml.Node, key string, values []string) {
	seqNode := yaml.Node{Kind: yaml.SequenceNode}
	for _, v := range values {
		seqNode.Content = append(seqNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: v})
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&seqNode,
	)
}

func addMapField(node *yaml.Node, key string, values map[string]string) {
	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	// Sort exits in a logical order
	sort.Slice(keys, func(i, j int) bool {
		order := map[string]int{"north": 0, "south": 1, "east": 2, "west": 3, "portal": 4}
		oi, ok1 := order[keys[i]]
		oj, ok2 := order[keys[j]]
		if !ok1 {
			oi = 99
		}
		if !ok2 {
			oj = 99
		}
		return oi < oj
	})

	mapNode := yaml.Node{Kind: yaml.MappingNode}
	for _, k := range keys {
		mapNode.Content = append(mapNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: values[k]},
		)
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&mapNode,
	)
}
