package labyrinth

import (
	"fmt"
	"os"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
	"gopkg.in/yaml.v3"
)

// LabyrinthConfig represents the structure of the labyrinth YAML file
type LabyrinthConfig struct {
	Width         int                       `yaml:"width"`
	Height        int                       `yaml:"height"`
	GeneratedSeed int64                     `yaml:"generated_seed"`
	Gates         []GateConfigYAML          `yaml:"gates"`
	Shortcuts     []ShortcutConfigYAML      `yaml:"shortcuts"`
	Rooms         map[string]RoomConfigYAML `yaml:"rooms"`
}

// GateConfigYAML represents a gate in the YAML config
type GateConfigYAML struct {
	CityID   string `yaml:"city_id"`
	CityName string `yaml:"city_name"`
	RoomID   string `yaml:"room_id"`
}

// ShortcutConfigYAML represents a shortcut pair in the YAML config
type ShortcutConfigYAML struct {
	RoomA string `yaml:"room_a"`
	RoomB string `yaml:"room_b"`
}

// RoomConfigYAML represents a room in the YAML config
type RoomConfigYAML struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"`
	Features    []string          `yaml:"features"`
	Exits       map[string]string `yaml:"exits"`
}

// LoadFromYAML loads the labyrinth from a YAML file
func LoadFromYAML(filename string) (*Labyrinth, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read labyrinth file: %w", err)
	}

	var config LabyrinthConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse labyrinth YAML: %w", err)
	}

	return CreateLabyrinth(&config)
}

// CreateLabyrinth creates a labyrinth from a configuration
func CreateLabyrinth(config *LabyrinthConfig) (*Labyrinth, error) {
	if config == nil || len(config.Rooms) == 0 {
		return nil, fmt.Errorf("labyrinth configuration is empty")
	}

	lab := New()
	lab.Width = config.Width
	lab.Height = config.Height

	// Store gate info
	for _, gate := range config.Gates {
		lab.Gates = append(lab.Gates, GateInfo{
			CityID:   gate.CityID,
			CityName: gate.CityName,
			RoomID:   gate.RoomID,
		})
		lab.GateRooms[gate.CityID] = gate.RoomID
	}

	// Store shortcut info
	for _, sc := range config.Shortcuts {
		lab.Shortcuts = append(lab.Shortcuts, ShortcutInfo{
			RoomA: sc.RoomA,
			RoomB: sc.RoomB,
		})
	}

	// First pass: create all rooms
	for roomID, def := range config.Rooms {
		roomType := parseLabyrinthRoomType(def.Type)

		room := world.NewRoom(roomID, def.Name, def.Description, roomType)
		room.Floor = -1 // Labyrinth has no floor number (use -1 to indicate labyrinth)

		// Add features
		for _, feature := range def.Features {
			room.AddFeature(feature)
		}

		lab.addRoom(room)
	}

	// Second pass: link exits
	for roomID, def := range config.Rooms {
		room := lab.GetRoom(roomID)
		if room == nil {
			continue
		}

		for direction, targetID := range def.Exits {
			target := lab.GetRoom(targetID)
			if target == nil {
				// Skip invalid exits (they'll be connected to cities later)
				continue
			}
			room.AddExit(direction, target)
		}
	}

	return lab, nil
}

// parseLabyrinthRoomType converts a string to the appropriate room type
func parseLabyrinthRoomType(s string) world.RoomType {
	switch s {
	case "labyrinth_gate":
		return world.RoomTypeLabyrinthGate
	case "labyrinth":
		return world.RoomTypeLabyrinth
	default:
		return world.RoomTypeLabyrinth
	}
}

// ConnectGateToCity connects a labyrinth gate to a city room
// The gateRoom is the labyrinth gate, and cityGateRoom is the city's gate room
func ConnectGateToCity(gateRoom, cityGateRoom *world.Room, labyrinthDirection, cityDirection string) {
	if gateRoom == nil || cityGateRoom == nil {
		return
	}

	// Connect labyrinth gate to city gate room
	gateRoom.AddExit(labyrinthDirection, cityGateRoom)

	// Connect city gate room back to labyrinth
	cityGateRoom.AddExit(cityDirection, gateRoom)
}
