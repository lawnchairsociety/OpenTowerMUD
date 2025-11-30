package tower

import (
	"fmt"
	"os"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
	"gopkg.in/yaml.v3"
)

// CityRoomDef represents a room definition from the city_rooms.yaml file
type CityRoomDef struct {
	Name             string            `yaml:"name"`
	Description      string            `yaml:"description"`
	DescriptionDay   string            `yaml:"description_day"`
	DescriptionNight string            `yaml:"description_night"`
	Type             string            `yaml:"type"`
	Features         []string          `yaml:"features"`
	Exits            map[string]string `yaml:"exits"` // direction -> room_id
}

// CityConfig represents the structure of the city_rooms.yaml file
type CityConfig struct {
	Rooms map[string]CityRoomDef `yaml:"rooms"`
}

// LoadCityFromYAML loads the city configuration from a YAML file
func LoadCityFromYAML(filename string) (*CityConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read city rooms file: %w", err)
	}

	var config CityConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse city rooms YAML: %w", err)
	}

	return &config, nil
}

// CreateCityFloor creates the ground floor (floor 0) from a city configuration
func CreateCityFloor(config *CityConfig) (*Floor, error) {
	if config == nil || len(config.Rooms) == 0 {
		return nil, fmt.Errorf("city configuration is empty")
	}

	floor := NewFloor(0) // Floor 0 is the city

	// First pass: create all rooms
	for roomID, def := range config.Rooms {
		roomType := world.RoomTypeCity // Default to city type

		room := world.NewRoom(roomID, def.Name, def.Description, roomType)
		room.Floor = 0
		room.DescriptionDay = def.DescriptionDay
		room.DescriptionNight = def.DescriptionNight

		// Add features
		for _, feature := range def.Features {
			room.AddFeature(feature)
		}

		floor.AddRoom(room)

		// Track special rooms
		if hasFeature(def.Features, "portal") {
			floor.SetPortalRoom(roomID)
		}
		if hasFeature(def.Features, "stairs_up") {
			floor.SetStairsUp(roomID) // City stairs go up to tower
		}
	}

	// Second pass: link exits
	for roomID, def := range config.Rooms {
		room := floor.GetRoom(roomID)
		if room == nil {
			continue
		}

		for direction, targetID := range def.Exits {
			target := floor.GetRoom(targetID)
			if target == nil {
				// Allow "up" and "down" exits to have no target - they're resolved dynamically
				if direction == "up" || direction == "down" {
					room.AddExit(direction, nil)
					continue
				}
				return nil, fmt.Errorf("room %q has exit to non-existent room %q", roomID, targetID)
			}
			room.AddExit(direction, target)
		}
	}

	return floor, nil
}

// LoadAndCreateCity loads the city from a YAML file and creates the floor
func LoadAndCreateCity(filename string) (*Floor, error) {
	config, err := LoadCityFromYAML(filename)
	if err != nil {
		return nil, err
	}

	return CreateCityFloor(config)
}

// GetSpawnRoom returns the room ID where new players should spawn
func GetSpawnRoom() string {
	return "town_square"
}

// GetTowerEntranceRoom returns the room ID of the tower entrance
func GetTowerEntranceRoom() string {
	return "tower_entrance"
}

// hasFeature checks if a feature list contains a specific feature
func hasFeature(features []string, target string) bool {
	for _, f := range features {
		if f == target {
			return true
		}
	}
	return false
}

// ValidateCityFloor checks that the city floor has required rooms and features
func ValidateCityFloor(floor *Floor) error {
	if floor == nil {
		return fmt.Errorf("floor is nil")
	}

	if floor.Number != 0 {
		return fmt.Errorf("city floor must be floor 0, got %d", floor.Number)
	}

	// Check for spawn room
	spawnRoom := floor.GetRoom(GetSpawnRoom())
	if spawnRoom == nil {
		return fmt.Errorf("missing spawn room: %s", GetSpawnRoom())
	}

	// Check for portal in spawn room
	if !spawnRoom.HasFeature("portal") {
		return fmt.Errorf("spawn room missing portal feature")
	}

	// Check for tower entrance
	entranceRoom := floor.GetRoom(GetTowerEntranceRoom())
	if entranceRoom == nil {
		return fmt.Errorf("missing tower entrance room: %s", GetTowerEntranceRoom())
	}

	// Check for stairs in tower entrance
	if !entranceRoom.HasFeature("stairs_up") {
		return fmt.Errorf("tower entrance missing stairs_up feature")
	}

	// Check floor has stairs up set
	if floor.StairsUpRoom == "" {
		return fmt.Errorf("city floor missing StairsUpRoom")
	}

	// Check floor has portal set
	if floor.PortalRoom == "" {
		return fmt.Errorf("city floor missing PortalRoom")
	}

	return nil
}
