package tower

import (
	"fmt"
	"os"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
	"gopkg.in/yaml.v3"
)

// FloorYAML represents a floor loaded from YAML
type FloorYAML struct {
	Floor         int                  `yaml:"floor"`
	Tower         string               `yaml:"tower"`
	GeneratedSeed int64                `yaml:"generated_seed"`
	StairsUp      string               `yaml:"stairs_up"`
	StairsDown    string               `yaml:"stairs_down"`
	PortalRoom    string               `yaml:"portal_room"`
	Rooms         map[string]*RoomYAML `yaml:"rooms"`
}

// RoomYAML represents a room loaded from YAML
type RoomYAML struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"`
	Features    []string          `yaml:"features"`
	Exits       map[string]string `yaml:"exits"`
}

// LoadFloorFromYAML loads a floor from a YAML file
func LoadFloorFromYAML(path string) (*Floor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read floor file: %w", err)
	}

	var floorYAML FloorYAML
	if err := yaml.Unmarshal(data, &floorYAML); err != nil {
		return nil, fmt.Errorf("failed to parse floor YAML: %w", err)
	}

	return floorYAML.ToFloor()
}

// ToFloor converts the YAML representation to a Floor with world.Room objects
func (fy *FloorYAML) ToFloor() (*Floor, error) {
	floor := NewFloor(fy.Floor)
	floor.GeneratedSeed = fy.GeneratedSeed

	// First pass: create all rooms
	for roomID, roomYAML := range fy.Rooms {
		roomType := parseRoomType(roomYAML.Type)
		room := world.NewRoom(roomID, roomYAML.Name, roomYAML.Description, roomType)
		room.Floor = fy.Floor

		// Add features
		for _, feature := range roomYAML.Features {
			room.AddFeature(feature)
		}

		floor.AddRoom(room)
	}

	// Second pass: link exits
	for roomID, roomYAML := range fy.Rooms {
		room := floor.GetRoom(roomID)
		if room == nil {
			continue
		}

		for dir, targetID := range roomYAML.Exits {
			targetRoom := floor.GetRoom(targetID)
			if targetRoom == nil {
				// Exit points to a room not on this floor - this is expected for stairs
				// The tower will link these when connecting floors
				continue
			}
			room.AddExit(dir, targetRoom)
		}
	}

	// Set special room references
	if fy.StairsUp != "" {
		floor.SetStairsUp(fy.StairsUp)
	}
	if fy.StairsDown != "" {
		floor.SetStairsDown(fy.StairsDown)
	}
	if fy.PortalRoom != "" {
		floor.SetPortalRoom(fy.PortalRoom)
	}

	return floor, nil
}

// parseRoomType converts a string room type to world.RoomType
func parseRoomType(typeStr string) world.RoomType {
	switch typeStr {
	case "corridor":
		return world.RoomTypeCorridor
	case "room":
		return world.RoomTypeRoom
	case "dead_end":
		return world.RoomTypeRoom // Dead ends are just rooms
	case "stairs_up", "stairs_down":
		return world.RoomTypeStairs
	case "treasure":
		return world.RoomTypeTreasure
	case "boss":
		return world.RoomTypeBoss
	default:
		return world.RoomTypeRoom
	}
}

// FloorFileExists checks if a floor YAML file exists
func FloorFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
